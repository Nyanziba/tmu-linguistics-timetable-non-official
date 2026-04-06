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
// マッチした科目にはpriority, level, recommended_yearを付与して返す。
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
			Level:           determineLevelFromDepartmentCode(scraped.DepartmentCode),
			RecommendedYear: 0,
		}

		if bestMatch := findBestMatch(scraped, requiredCourses); bestMatch != nil {
			matched.Priority = bestMatch.Priority
			matched.RecommendedYear = bestMatch.RecommendedYear
			matched.MaxRecommendedYear = bestMatch.MaxRecommendedYear
			if matched.MaxRecommendedYear == 0 {
				matched.MaxRecommendedYear = matched.RecommendedYear
			}
			// Level がrequiredCourse側で明示されていればそちらを優先
			if bestMatch.Level != "" {
				matched.Level = bestMatch.Level
			}
		}

		if matched.Priority != "" {
			matchedCourses = append(matchedCourses, matched)
		}
	}

	return matchedCourses
}

// determineLevelFromDepartmentCode は学部コードから「学部」or「大学院」を判定する。
func determineLevelFromDepartmentCode(departmentCode string) string {
	// A1=人文社会学部(学部), 11=人文科学研究科(大学院)
	switch departmentCode {
	case "A1":
		return "学部"
	case "11":
		return "大学院"
	default:
		return "学部"
	}
}

// findBestMatch はスクレイプした科目に最もマッチする必修科目を探す。
func findBestMatch(scraped model.ScrapedCourse, requiredCourses []model.RequiredCourse) *model.RequiredCourse {
	// 1. 授業番号での正確マッチを最優先
	for index := range requiredCourses {
		required := &requiredCourses[index]
		if required.CourseCode != "" && required.CourseCode == scraped.CourseCode {
			return required
		}
	}

	// 2. コードプレフィックスでのマッチ
	for index := range requiredCourses {
		required := &requiredCourses[index]
		if required.CourseCodePrefix != "" && strings.HasPrefix(scraped.CourseCode, required.CourseCodePrefix) {
			return required
		}
	}

	// 3. 科目名+教員名の部分一致でマッチ
	normalizedCourseName := normalizeCourseNameForMatching(scraped.CourseName)
	normalizedInstructor := normalizeInstructorForMatching(scraped.Instructor)

	for index := range requiredCourses {
		required := &requiredCourses[index]
		if required.CourseCode != "" || required.CourseCodePrefix != "" {
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

func normalizeCourseNameForMatching(name string) string {
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

	result = strings.Join(strings.Fields(result), " ")
	return result
}

func normalizeInstructorForMatching(instructor string) string {
	instructor = strings.ReplaceAll(instructor, "\u3000", " ")
	instructor = strings.TrimSpace(instructor)
	return instructor
}
