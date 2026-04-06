package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"tmu-timetable/matcher"
	"tmu-timetable/model"
	"tmu-timetable/scraper"
)

// スクレイプ対象の学部コード
var departmentCodes = []string{
	"A1", // 人文社会学部
	"11", // 人文科学研究科（大学院）
}

func main() {
	year := flag.Int("year", 2026, "対象年度")
	outputPath := flag.String("output", "src/data/courses.json", "出力先JSONファイルパス")
	flag.Parse()

	var allScrapedCourses []model.ScrapedCourse
	for _, departmentCode := range departmentCodes {
		fmt.Printf("シラバスサイトから %d年度 %s の科目を取得中...\n", *year, departmentCode)
		courses, err := scraper.FetchAllCourses(*year, departmentCode)
		if err != nil {
			fmt.Fprintf(os.Stderr, "スクレイプに失敗 (%s): %v\n", departmentCode, err)
			os.Exit(1)
		}
		fmt.Printf("  %s: %d科目\n", departmentCode, len(courses))
		allScrapedCourses = append(allScrapedCourses, courses...)
	}
	fmt.Printf("取得合計: %d科目\n", len(allScrapedCourses))

	requiredCourses, err := matcher.LoadRequiredCourses()
	if err != nil {
		fmt.Fprintf(os.Stderr, "必修科目の読み込みに失敗: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("必修科目マスター: %d件\n", len(requiredCourses))

	matchedCourses := matcher.MatchCourses(allScrapedCourses, requiredCourses)
	fmt.Printf("マッチした科目: %d件\n", len(matchedCourses))

	output := model.CoursesOutput{
		Year:        *year,
		Semester:    "全期",
		GeneratedAt: time.Now().Format(time.RFC3339),
		Courses:     matchedCourses,
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "JSON生成に失敗: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*outputPath, jsonData, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "ファイル書き込みに失敗: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("出力完了: %s\n", *outputPath)

	for _, course := range matchedCourses {
		fmt.Printf("  [%s][%s] %s %s %s%d限 %s\n",
			course.Priority,
			course.Level,
			course.Semester,
			course.DayOfWeek,
			padRight(course.CourseName, 24),
			course.Period,
			course.Instructor,
		)
	}
}

func padRight(text string, width int) string {
	runeCount := 0
	for _, r := range text {
		if r > 0x7F {
			runeCount += 2
		} else {
			runeCount++
		}
	}
	if runeCount >= width {
		return text
	}
	return text + fmt.Sprintf("%*s", width-runeCount, "")
}
