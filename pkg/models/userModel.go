package models

import (
	"gorm.io/gorm"
	"time"
)

type User struct {
	gorm.Model
	Name        string
	Secondname  string
	Vuz         string       `gorm:"type:vuz;default:'не указано'"`
	Kafedra     string       `gorm:"type:kafedra;default:'не указано'"`
	Fakultet    string       `gorm:"type:fakultet;default:'не указано'"`
	Email       string       `gorm:"unique;not null"`
	Password    string       `gorm:"not null"`
	Role        string       `gorm:"type:user_role;default:'студент'"`
	UserCourses []UserCourse `gorm:"foreignKey:UserID"`
	TestResults []TestResult `gorm:"foreignKey:UserID"`
}

type Course struct {
	gorm.Model
	Name        string `gorm:"unique;not null"`
	Description string
	Category    string       `gorm:"type:varchar(100)"` // enum
	Tests       []Test       `gorm:"foreignKey:CourseID"`
	Documents   []Documenti  `gorm:"foreignKey:CourseID"`
	UserCourses []UserCourse `gorm:"foreignKey:CourseID"`
	Lessonis    []Lessons    `gorm:"foreignKey:CourseID"`
	Videos      []TestVideo  `gorm:"foreignKey:CourseID"`
}

type UserCourse struct {
	gorm.Model
	UserID   uint   `gorm:"index;not null"`
	CourseID uint   `gorm:"index;not null"`
	Status   string `gorm:"type:user_course_status;default:'не пройден'"`
}

type Lessons struct {
	gorm.Model
	CourseID    uint   `gorm:"index;not null"`
	Title       string `gorm:"not null"`
	Description string
	Order       int
	Tests       []Test      `gorm:"foreignKey:LessonID"`
	Documents   []Documenti `gorm:"foreignKey:LessonID"`
	Videos      []TestVideo `gorm:"foreignKey:LessonID"`
}

type Test struct {
	gorm.Model
	CourseID    uint   `gorm:"index;not null"`
	LessonID    uint   `gorm:"index;not null"`
	Title       string `gorm:"not null"`
	Description string
	PassMark    int `gorm:"not null"`
	MaxAttempts int `gorm:"default:1"`
	Duration    int `gorm:"not null" json:"duration"`
	Questions   []TestQuestion
	Results     []TestResult
}

type TestQuestion struct {
	gorm.Model
	TestID        uint   `gorm:"index;not null"`
	CourseID      uint   `gorm:"index;not null"`
	LessonID      uint   `gorm:"index;not null"`
	Question      string `gorm:"not null"`
	QuestionType  string `gorm:"type:question_type;not null" json:"question_type"`
	Order         int    `gorm:"not null"`
	CorrectAnswer string `gorm:"not null"`
}

type TestResult struct {
	gorm.Model
	UserID      uint `gorm:"index;not null"`
	TestID      uint `gorm:"index;not null"`
	Score       int
	Status      string `gorm:"type:test_result_status;default:'не выполнено'"`
	Attempt     int
	StartedAt   time.Time
	CompletedAt time.Time
}

type Documenti struct {
	gorm.Model
	CourseID uint    `gorm:"index;not null"`
	LessonID uint    `gorm:"index;not null"`
	FileName string  `gorm:"not null"`
	FileType string  `gorm:"not null"`
	FilePath string  `gorm:"not null"`
	FileSize float64 `gorm:"not null"`
}

type TestVideo struct {
	gorm.Model
	CourseID uint    `gorm:"index;not null"`
	LessonID uint    `gorm:"index;not null"`
	FileName string  `gorm:"not null"`
	FileType string  `gorm:"not null"`
	FilePath string  `gorm:"not null"`
	FileSize float64 `gorm:"not null"`
}

type ResponseChat struct {
	gorm.Model
	UserID   uint `gorm:"index;not null"`
	Answer   string
	Response any
}

type ResponseSummary struct {
	gorm.Model
	VideoID uint `gorm:"index;not null"`
	Summary any
}
