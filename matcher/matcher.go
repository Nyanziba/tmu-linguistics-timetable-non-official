package matcher

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"tmu-timetable/model"
)

//go:embed required_courses.json
var requiredCoursesJSON []byte

// LoadRequiredCourses は埋め込まれた必修科目JSONを読み込む。
func LoadRequiredCourses() ([]model.RequiredCourse, error) {
	var courses []model.RequiredCourse
	if err := json.Unmarshal(requiredCoursesJSON, &courses); err != nil {
		return nil, fmt.Errorf("必修科目JSONの読み込みに失敗: %w", err)
	}
	return courses, nil
}

// MatchCourses はスクレイプした科目と必修科目リストを照合し、
// マッチした科目にはpriorityとrecommended_yearを付与して返す。
func MatchCourses(scrapedCourses []model.ScrapedCourse, requiredCourses []model.RequiredCourse) []model.MatchedCourse {
	var matchedCourses []model.MatchedCourse

	for _, scraped := range scrapedCourses {
		matched := model.MatchedCourse{
			CourseName:      scraped.CourseName,
			CourseCode:      scraped.CourseCode,
			Instructor:      scraped.Instructor,
			DayOfWeek:       scraped.DayOfWeek,
			Period:          scraped.Period,
			Semester:        scraped.Semester,
			Credits:         scraped.Credits,
			DetailURL:       scraped.DetailURL,
			Priority:        "",
			RecommendedYear: 0,
		}

		if bestMatch := findBestMatch(scraped, requiredCourses); bestMatch != nil {
			matched.Priority = bestMatch.Priority
			matched.RecommendedYear = bestMatch.RecommendedYear
			matched.MaxRecommendedYear = bestMatch.MaxRecommendedYear
			if matched.MaxRecommendedYear == 0 {
				matched.MaxRecommendedYear = matched.RecommendedYear
			}
		}

		// 必修リストにマッチした科目のみ出力する
		if matched.Priority != "" {
			matchedCourses = append(matchedCourses, matched)
		}
	}

	return matchedCourses
}

// findBestMatch はスクレイプした科目に最もマッチする必修科目を探す。
// CourseCodeが設定されている場合は授業番号で正確にマッチし、
// そうでなければ科目名の部分一致 + 教員名の部分一致で判定する。
func findBestMatch(scraped model.ScrapedCourse, requiredCourses []model.RequiredCourse) *model.RequiredCourse {
	// まず授業番号での正確マッチを優先
	for index := range requiredCourses {
		required := &requiredCourses[index]
		if required.CourseCode != "" && required.CourseCode == scraped.CourseCode {
			return required
		}
	}

	// 次に科目名+教員名の部分一致でマッチ
	normalizedCourseName := normalizeCourseNameForMatching(scraped.CourseName)
	normalizedInstructor := normalizeInstructorForMatching(scraped.Instructor)

	for index := range requiredCourses {
		required := &requiredCourses[index]
		// CourseCodeが設定されている場合は部分一致では使わない
		if required.CourseCode != "" {
			continue
		}

		normalizedRequiredName := normalizeCourseNameForMatching(required.Name)
		normalizedRequiredInstructor := normalizeInstructorForMatching(required.Instructor)

		nameMatches := strings.Contains(normalizedCourseName, normalizedRequiredName)

		instructorMatches := normalizedRequiredInstructor == "" ||
			strings.Contains(normalizedInstructor, normalizedRequiredInstructor)

		if nameMatches && instructorMatches {
			return required
		}
	}

	return nil
}

// normalizeCourseNameForMatching は科目名を正規化してマッチングしやすくする。
// ローマ数字を半角英字に変換し、装飾的な括弧内の数字を除去する。
func normalizeCourseNameForMatching(name string) string {
	// 全角→半角の基本変換
	replacements := map[string]string{
		"Ⅰ": "I", "Ⅱ": "II", "Ⅲ": "III", "Ⅳ": "IV", "Ⅴ": "V",
		"ⅰ": "I", "ⅱ": "II", "ⅲ": "III", "ⅳ": "IV", "ⅴ": "V",
		"１": "1", "２": "2", "３": "3", "４": "4", "５": "5",
		"＜": "<", "＞": ">",
		"（": "(", "）": ")",
	}

	result := name
	for original, replacement := range replacements {
		result = strings.ReplaceAll(result, original, replacement)
	}

	// スペース正規化
	result = strings.Join(strings.Fields(result), " ")
	return result
}

// normalizeInstructorForMatching は教員名を正規化する。
// 姓だけ残して比較しやすくする。
func normalizeInstructorForMatching(instructor string) string {
	// 全角スペースを半角に
	instructor = strings.ReplaceAll(instructor, "\u3000", " ")
	instructor = strings.TrimSpace(instructor)

	// 「橋本（龍）」→「橋本」、「パク ウンビ」→「パク」のように姓を取得
	// ただし完全一致ではなく部分一致で使うので、そのまま返す
	return instructor
}
