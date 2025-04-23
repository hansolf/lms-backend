package lessons

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/segmentio/kafka-go"
	"lms-go/pkg/initial"
	"lms-go/pkg/kfka"
	"lms-go/pkg/middleware"
	"lms-go/pkg/models"
	"lms-go/pkg/search"
	"log"
	"net/http"
	"strconv"
)

func CreateLes(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}
	if user.Role != string(middleware.Admin) && user.Role != string(middleware.Teacher) {
		http.Error(w, "Доступ запрещен", http.StatusForbidden)
		return
	}
	vars := mux.Vars(r)
	courseID, err := strconv.Atoi(vars["courseID"])
	if err != nil {
		http.Error(w, "Некорректный courseID", http.StatusBadRequest)
		return
	}

	var lessons models.Lessons
	err = json.NewDecoder(r.Body).Decode(&lessons)
	if err != nil {
		http.Error(w, "Проблема с декодированием", http.StatusBadRequest)
		return
	}
	lessons.CourseID = uint(courseID)

	result := initial.DB.Preload("Documents").Preload("Tests").Create(&lessons)
	if result.Error != nil {
		http.Error(w, "Не удалось создать урок", http.StatusInternalServerError)
		return
	}
	go search.IndexLesson(lessons, lessons.ID)
	kafkaWriter := &kafka.Writer{
		Addr:     kafka.TCP("localhost:9092"),
		Topic:    "course_notifications",
		Balancer: &kafka.LeastBytes{},
	}
	defer kafkaWriter.Close()
	var userCourses []models.UserCourse
	initial.DB.Where("course_id = ?", courseID).Find(&userCourses)
	var course models.Course
	initial.DB.First(&course, courseID)
	for _, userCourse := range userCourses {
		var poluchatel models.User
		initial.DB.First(&poluchatel, userCourse.UserID)
		event := kfka.NotificationCourse{
			UserID:      userCourse.UserID,
			CourseID:    lessons.CourseID,
			LessonId:    lessons.ID,
			CourseName:  course.Name,
			Lesson:      lessons.Title,
			Description: lessons.Description,
			Email:       poluchatel.Email,
			EventType:   fmt.Sprintf("Добавлен новый урок: %v в курс %v", lessons.Title, course.Name),
		}
		if err = event.SendNotif(kafkaWriter); err != nil {
			log.Println("Ошибка отправки Kafka:", err)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lessons)
}

func GetLessons(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	courseID := vars["courseID"]

	var lessons []models.Lessons
	result := initial.DB.Where("course_id = ?", courseID).Preload("Documents").Preload("Tests").Find(&lessons)
	if result.Error != nil {
		http.Error(w, "Не удалось получить уроки курса", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lessons)
}

func GetLesson(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	lessonID := vars["id"]
	var lesson models.Lessons
	result := initial.DB.Preload("Tests").Preload("Documents").First(&lesson, lessonID)
	if result.Error != nil {
		http.Error(w, "Курс не найден", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lesson)
}

func UpdateLes(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	courseID := vars["courseID"]
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}
	if user.Role != string(middleware.Admin) && user.Role != string(middleware.Teacher) {
		http.Error(w, "Доступ запрещен", http.StatusForbidden)
		return
	}
	var lesson models.Lessons
	result := initial.DB.Preload("Documents").Preload("Tests").Where("course_id = ?", courseID).First(&lesson, id)
	if result.Error != nil {
		http.Error(w, "Не удалось найти урок", http.StatusInternalServerError)
		return
	}
	var update models.Lessons
	err := json.NewDecoder(r.Body).Decode(&update)
	if err != nil {
		http.Error(w, "Ошибка с декодированием", http.StatusBadRequest)
		return
	}
	if update.Title != "" {
		lesson.Title = update.Title
	}
	if update.Description != "" {
		lesson.Description = update.Description
	}
	if len(update.Documents) != 0 {
		lesson.Documents = update.Documents
	}
	if len(update.Tests) != 0 {
		lesson.Tests = update.Tests
	}
	result = initial.DB.Preload("Documents").Preload("Tests").Save(&lesson)
	if result.Error != nil {
		http.Error(w, "Не удалось сохранить урок", http.StatusInternalServerError)
		return
	}
	go search.IndexLesson(lesson, lesson.ID)
	kafkawriter := &kafka.Writer{
		Addr:     kafka.TCP("localhost:9092"),
		Topic:    "course_update_notifications",
		Balancer: &kafka.LeastBytes{},
	}
	defer kafkawriter.Close()
	var userCourses []models.UserCourse
	initial.DB.Where("course_id = ?", courseID).Find(&userCourses)
	var course models.Course
	initial.DB.First(&course, courseID)
	for _, userCourse := range userCourses {
		var poluchatel models.User
		initial.DB.First(&poluchatel, userCourse.UserID)
		zap := kfka.NotificationCourse{
			UserID:      userCourse.UserID,
			CourseID:    lesson.CourseID,
			LessonId:    lesson.ID,
			CourseName:  course.Name,
			Email:       poluchatel.Email,
			Lesson:      lesson.Title,
			Description: lesson.Description,
			EventType:   fmt.Sprintf("Урок: %v из курса %v обновлен", lesson.Title, course.Name),
		}
		err = zap.SendNotif(kafkawriter)
		if err != nil {
			log.Println("Ошибка отправки Kafka:", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lesson)
}

func DeleteLesson(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}
	if user.Role != string(middleware.Admin) && user.Role != string(middleware.Teacher) {
		http.Error(w, "Нет доступа", http.StatusForbidden)
		return
	}
	initial.DB.Where("lessons_id = ?", id).Delete(&models.Test{})
	initial.DB.Where("lessons_id = ?", id).Delete(&models.Documenti{})

	result := initial.DB.Delete(&models.Lessons{}, id)
	if result.Error != nil {
		http.Error(w, "Ошибка при удалении урока", http.StatusInternalServerError)
		return
	}
	go search.DeleteLessonFromIndex(parseUint(id))
	w.WriteHeader(http.StatusOK)
}

func parseUint(s string) uint {
	u, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return uint(u)
}
