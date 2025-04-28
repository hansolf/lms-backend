package tests

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"lms-go/pkg/initial"
	"lms-go/pkg/middleware"
	"lms-go/pkg/models"
	"net/http"
	"strconv"
)

func CreateQ(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}
	if user.Role != string(middleware.Admin) && user.Role != string(middleware.Teacher) {
		http.Error(w, "Нет прав", http.StatusForbidden)
		return
	}
	vars := mux.Vars(r)
	courseID, _ := strconv.Atoi(vars["courseID"])
	lessonsID, _ := strconv.Atoi(vars["lessonsID"])
	testID, _ := strconv.Atoi(vars["testID"])
	var testquestion models.TestQuestion
	err := json.NewDecoder(r.Body).Decode(&testquestion)
	if err != nil {
		http.Error(w, "Проблема с декодированием", http.StatusBadRequest)
		return
	}
	testquestion.TestID = uint(testID)
	testquestion.LessonID = uint(lessonsID)
	testquestion.CourseID = uint(courseID)
	result := initial.DB.Create(&testquestion)
	if result.Error != nil {
		http.Error(w, "Не удалось создать вопрос", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(testquestion)
}

func DeleteQ(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}
	if user.Role != string(middleware.Admin) && user.Role != string(middleware.Teacher) {
		http.Error(w, "Нет прав", http.StatusForbidden)
		return
	}
	vars := mux.Vars(r)
	quesID := vars["id"]
	result := initial.DB.Delete(&models.TestQuestion{}, quesID)
	if result.Error != nil {
		http.Error(w, "Не удалось удалить вопрос", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func GetQ(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	quesID := vars["id"]
	var testquestion models.TestQuestion
	result := initial.DB.First(&testquestion, quesID)
	if result.Error != nil {
		http.Error(w, "Не удалось найти вопрос", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(testquestion)
}

/*	TestID        uint   `gorm:"index;not null"`
	Question      string `gorm:"not null"`
	QuestionType  string `gorm:"type:question_type;not null"`
	Order         int    `gorm:"not null"`
	CorrectAnswer string `gorm:"not null"`*/

func UpdateQ(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}
	if user.Role != string(middleware.Admin) && user.Role != string(middleware.Teacher) {
		http.Error(w, "Нет прав", http.StatusForbidden)
		return
	}
	vars := mux.Vars(r)
	quesID := vars["id"]
	var testquestion models.TestQuestion
	result := initial.DB.First(&testquestion, quesID)
	if result.Error != nil {
		http.Error(w, "Не удалось найти тест", http.StatusInternalServerError)
		return
	}
	var update models.TestQuestion
	err := json.NewDecoder(r.Body).Decode(&update)
	if err != nil {
		http.Error(w, "Ошибка с декодированием", http.StatusBadRequest)
		return
	}
	if update.Question != testquestion.Question {
		testquestion.Question = update.Question
	}
	if update.QuestionType != testquestion.QuestionType {
		testquestion.QuestionType = update.QuestionType
	}
	if update.Order != testquestion.Order {
		testquestion.Order = update.Order
	}
	if update.CorrectAnswer != testquestion.CorrectAnswer {
		testquestion.CorrectAnswer = update.CorrectAnswer
	}
	result = initial.DB.Save(&testquestion)
	if result.Error != nil {
		http.Error(w, "Ошибка с сохранением вопроса", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(testquestion)
}

func GetQs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	testID := vars["testID"]
	var testquestions []models.TestQuestion
	result := initial.DB.Where("test_id = ?", testID).Find(&testquestions)
	if result.Error != nil {
		http.Error(w, "Не удалось найти вопросы по уроку", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(testquestions)
}
