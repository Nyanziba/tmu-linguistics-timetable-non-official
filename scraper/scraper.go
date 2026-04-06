package scraper

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/sync/errgroup"

	"tmu-timetable/model"
)

const (
	baseURL        = "https://kyouikujouhou.eas.tmu.ac.jp/syllabus"
	cacheDirectory = "cache"
)

// dayNumbers は曜日名とURL中の番号の対応。9は「他」（集中講義等）。
var dayNumbers = []struct {
	number    int
	dayOfWeek string
}{
	{1, "月"},
	{2, "火"},
	{3, "水"},
	{4, "木"},
	{5, "金"},
	{9, "他"},
}

var detailURLPattern = regexp.MustCompile(`OpenInfo\('([^']+)'\)`)

// FetchAllCourses は指定された年度・学部コードの全曜日ページを並行取得し、
// 前期・通年の科目のみを返す。
func FetchAllCourses(year int, departmentCode string) ([]model.ScrapedCourse, error) {
	type result struct {
		courses []model.ScrapedCourse
	}

	results := make([]result, len(dayNumbers))
	group := new(errgroup.Group)

	for index, day := range dayNumbers {
		index, day := index, day
		group.Go(func() error {
			url := fmt.Sprintf("%s/%d/YobiIchiran_%s_%d.html", baseURL, year, departmentCode, day.number)
			courses, err := fetchAndParseTimetablePage(url, day.dayOfWeek, departmentCode)
			if err != nil {
				return fmt.Errorf("%s曜日の取得に失敗: %w", day.dayOfWeek, err)
			}
			results[index] = result{courses: courses}
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, err
	}

	var allCourses []model.ScrapedCourse
	for _, resultItem := range results {
		allCourses = append(allCourses, resultItem.courses...)
	}

	return allCourses, nil
}

func fetchAndParseTimetablePage(url string, dayOfWeek string, departmentCode string) ([]model.ScrapedCourse, error) {
	htmlBytes, err := CachedFetch(url, cacheDirectory, 24*time.Hour)
	if err != nil {
		return nil, err
	}

	return parseTimetableHTML(string(htmlBytes), dayOfWeek, departmentCode)
}

func parseTimetableHTML(htmlContent string, dayOfWeek string, departmentCode string) ([]model.ScrapedCourse, error) {
	document, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("HTMLパースに失敗: %w", err)
	}

	var courses []model.ScrapedCourse
	tableRows := findAllTableRows(document)

	for _, row := range tableRows {
		cells := extractTableCells(row)
		if len(cells) < 9 {
			continue
		}

		// ヘッダー行をスキップ（最初のセルが数字でない場合）
		rowNumber := strings.TrimSpace(cells[0].textContent)
		if _, err := strconv.Atoi(rowNumber); err != nil {
			continue
		}

		period := extractPeriodNumber(strings.TrimSpace(cells[4].textContent))
		credits := parseCredits(strings.TrimSpace(cells[8].textContent))
		detailURL := extractDetailURL(cells[6].onclickContent)

		course := model.ScrapedCourse{
			CourseName: strings.TrimSpace(cells[6].textContent),
			CourseCode: strings.TrimSpace(cells[7].textContent),
			Instructor: normalizeWhitespace(strings.TrimSpace(cells[5].textContent)),
			DayOfWeek:  strings.TrimSpace(cells[3].textContent),
			Period:     period,
			Semester:   strings.TrimSpace(cells[2].textContent),
			Credits:    credits,
			DetailURL:  detailURL,
		}

		// 「他」の曜日で曜日が空の場合は集中講義
		if dayOfWeek == "他" && course.DayOfWeek == "" {
			course.Semester = "集中"
		}

		courses = append(courses, course)
	}

	return courses, nil
}

type tableCell struct {
	textContent    string
	onclickContent string // onclick属性の値（リンクセルから取得）
}

func findAllTableRows(node *html.Node) []*html.Node {
	var rows []*html.Node
	var traverse func(*html.Node)
	traverse = func(currentNode *html.Node) {
		if currentNode.Type == html.ElementNode && currentNode.Data == "tr" {
			rows = append(rows, currentNode)
		}
		for child := currentNode.FirstChild; child != nil; child = child.NextSibling {
			traverse(child)
		}
	}
	traverse(node)
	return rows
}

func extractTableCells(row *html.Node) []tableCell {
	var cells []tableCell
	for child := row.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && (child.Data == "td" || child.Data == "th") {
			cells = append(cells, tableCell{
				textContent:    extractTextContent(child),
				onclickContent: findOnclickAttribute(child),
			})
		}
	}
	return cells
}

// findOnclickAttribute はノードツリー内のa要素からonclick属性値を取得する。
func findOnclickAttribute(node *html.Node) string {
	var result string
	var traverse func(*html.Node)
	traverse = func(currentNode *html.Node) {
		if currentNode.Type == html.ElementNode && currentNode.Data == "a" {
			for _, attr := range currentNode.Attr {
				if attr.Key == "onclick" {
					result = attr.Val
					return
				}
			}
		}
		for child := currentNode.FirstChild; child != nil; child = child.NextSibling {
			traverse(child)
			if result != "" {
				return
			}
		}
	}
	traverse(node)
	return result
}

func extractTextContent(node *html.Node) string {
	var textBuilder strings.Builder
	var traverse func(*html.Node)
	traverse = func(currentNode *html.Node) {
		if currentNode.Type == html.TextNode {
			textBuilder.WriteString(currentNode.Data)
		}
		for child := currentNode.FirstChild; child != nil; child = child.NextSibling {
			traverse(child)
		}
	}
	traverse(node)
	return textBuilder.String()
}

func extractDetailURL(onclickValue string) string {
	matches := detailURLPattern.FindStringSubmatch(onclickValue)
	if len(matches) < 2 {
		return ""
	}
	return fmt.Sprintf("%s/2026/%s", baseURL, matches[1])
}

func extractPeriodNumber(periodText string) int {
	// "1" や "1限" のような文字列から数値を抽出
	periodText = strings.TrimSuffix(periodText, "限")
	periodText = strings.TrimSpace(periodText)
	number, err := strconv.Atoi(periodText)
	if err != nil {
		return 0
	}
	return number
}

func parseCredits(creditsText string) float64 {
	creditsText = strings.TrimSpace(creditsText)
	value, err := strconv.ParseFloat(creditsText, 64)
	if err != nil {
		return 0
	}
	return value
}

func normalizeWhitespace(text string) string {
	// 全角スペースや連続スペースを通常のスペース1つに正規化
	text = strings.ReplaceAll(text, "\u3000", " ")
	text = strings.ReplaceAll(text, "\t", " ")
	text = strings.ReplaceAll(text, "\n", " ")
	fields := strings.Fields(text)
	return strings.Join(fields, " ")
}
