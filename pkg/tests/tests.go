package tests

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/segmentio/kafka-go"
	"lms-go/pkg/initial"
	"lms-go/pkg/kfka"
	"lms-go/pkg/middleware"
	"lms-go/pkg/models"
	"log"
	"net/http"
	"os"
	"strconv"
)

func CreateTest(w http.ResponseWriter, r *http.Request) {
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
	lessonID, _ := strconv.Atoi(vars["lessonID"])
	var tests models.Test
	err := json.NewDecoder(r.Body).Decode(&tests)
	if err != nil {
		http.Error(w, "Проблема с декодированием", http.StatusBadRequest)
		return
	}
	tests.CourseID = uint(courseID)
	tests.LessonID = uint(lessonID)
	result := initial.DB.Create(&tests)
	if result.Error != nil {
		http.Error(w, "Не удалось создать таблицу", http.StatusInternalServerError)
		return
	}
	kafkawriter := &kafka.Writer{
		Addr:     kafka.TCP(os.Getenv("KAFKA_ADDRESS")),
		Topic:    "test_notifications",
		Balancer: &kafka.LeastBytes{},
	}
	defer kafkawriter.Close()
	var userCourse models.UserCourse
	initial.DB.Where("course_id = ?", courseID).First(&userCourse)
	var poluchatel models.User
	initial.DB.First(&poluchatel, userCourse.UserID)
	var course models.Course
	initial.DB.First(&course, courseID)
	var lesson models.Lessons
	initial.DB.First(&lesson, lessonID)
	event := kfka.NotificationTest{
		CourseID: tests.CourseID,
		LessonID: tests.LessonID,
		Email:    poluchatel.Email,
		Test:     tests.Title,
		Course:   course.Name,
		Lesson:   lesson.Title,
	}
	err = event.SendNotTest(kafkawriter)
	if err != nil {
		log.Println("Не удалось отправить сообщение Kafka: ", err)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tests)
}

func GetTests(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	lessonID := vars["lessonID"]
	var tests []models.Test
	result := initial.DB.Where("lesson_id = ?", lessonID).Preload("Questions").Preload("Results").Find(&tests)
	if result.Error != nil {
		http.Error(w, "Не удалось найти тесты", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tests)
}

func GetTest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	testID := vars["id"]
	var test models.Test
	result := initial.DB.Preload("Questions").Preload("Results").First(&test, testID)
	if result.Error != nil {
		http.Error(w, "Не удалось найти тест", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(test)
}

func DeleteTest(w http.ResponseWriter, r *http.Request) {
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
	testID := vars["id"]

	initial.DB.Where("test_id = ?", testID).Delete(&models.TestQuestion{})
	initial.DB.Where("test_id = ?", testID).Delete(&models.TestResult{})

	result := initial.DB.Delete(&models.Test{}, testID)
	if result.Error != nil {
		http.Error(w, "Не удалось удалить тест", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func UpdateTest(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}
	if user.Role != string(middleware.Admin) && user.Role != string(middleware.Teacher) {
		http.Error(w, "Нет прав", http.StatusForbidden)
	}
	vars := mux.Vars(r)
	testID := vars["id"]

	var test models.Test
	result := initial.DB.Preload("Questions").Preload("Results").First(&test, testID)
	if result.Error != nil {
		http.Error(w, "Не удалось найти тест", http.StatusInternalServerError)
		return
	}
	var update models.Test
	err := json.NewDecoder(r.Body).Decode(&update)
	if err != nil {
		http.Error(w, "Ошибка с декодированием", http.StatusBadRequest)
		return
	}

	if update.Title != "" {
		test.Title = update.Title
	}
	if update.Description != "" {
		test.Description = update.Description
	}
	if update.MaxAttempts != 0 {
		test.MaxAttempts = update.MaxAttempts
	}
	if update.PassMark != 0 {
		test.PassMark = update.PassMark
	}
	result = initial.DB.Save(&test)
	if result.Error != nil {
		http.Error(w, "Не удалось сохранить тест", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(test)
}
