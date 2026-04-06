package model

// ScrapedCourse はシラバスサイトの時間割ページから取得した1科目の情報を表す。
type ScrapedCourse struct {
	CourseName     string  `json:"course_name"`
	CourseCode     string  `json:"course_code"`
	Instructor     string  `json:"instructor"`
	DayOfWeek      string  `json:"day_of_week"` // "月","火","水","木","金"
	Period         int     `json:"period"`       // 1-6, 0=集中
	Semester       string  `json:"semester"`     // "前期","後期","通年","集中"
	Credits        float64 `json:"credits"`
	DetailURL      string  `json:"detail_url"`
	DepartmentCode string  `json:"department_code"`
}

// RequiredCourse は言語科学教室の必修科目マスターの1エントリを表す。
// CourseCodeが設定されている場合は授業番号で正確にマッチし、
// CourseCodePrefixが設定されている場合はコードの前方一致でマッチし、
// いずれも空の場合は科目名+教員名の部分一致でマッチする。
type RequiredCourse struct {
	Name               string `json:"name"`
	CourseCode         string `json:"course_code"`
	CourseCodePrefix   string `json:"course_code_prefix"`
	Instructor         string `json:"instructor"`
	Priority           string `json:"priority"` // "A","B","C"
	Level              string `json:"level"`     // "学部","大学院"
	RecommendedYear    int    `json:"recommended_year"`
	MaxRecommendedYear int    `json:"max_recommended_year"`
	Note               string `json:"note"`
}

// MatchedCourse はスクレイプ結果と必修情報を統合した科目情報を表す。
type MatchedCourse struct {
	CourseName         string  `json:"course_name"`
	CourseCode         string  `json:"course_code"`
	Instructor         string  `json:"instructor"`
	DayOfWeek          string  `json:"day_of_week"`
	Period             int     `json:"period"`
	Semester           string  `json:"semester"`
	Credits            float64 `json:"credits"`
	DetailURL          string  `json:"detail_url"`
	Priority           string  `json:"priority"`
	Level              string  `json:"level"` // "学部","大学院"
	RecommendedYear    int     `json:"recommended_year"`
	MaxRecommendedYear int     `json:"max_recommended_year"`
}

// CoursesOutput はJSON出力のトップレベル構造を表す。
type CoursesOutput struct {
	Year        int             `json:"year"`
	Semester    string          `json:"semester"`
	GeneratedAt string          `json:"generated_at"`
	Courses     []MatchedCourse `json:"courses"`
}
