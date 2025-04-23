package courses

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"lms-go/pkg/initial"
	"lms-go/pkg/middleware"
	"lms-go/pkg/models"
	"lms-go/pkg/search"
	"net/http"
	"strconv"
)

func Create(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Пользователь не авторизован", http.StatusUnauthorized)
		return
	}

	if user.Role != string(middleware.Admin) && user.Role != string(middleware.Teacher) {
		http.Error(w, "Доступ запрещен", http.StatusForbidden)
		return
	}
	var course models.Course
	err := json.NewDecoder(r.Body).Decode(&course)
	if err != nil {
		http.Error(w, "Ошибка с декодированием", http.StatusBadRequest)
		return
	}
	result := initial.DB.Preload("Tests").Preload("Documents").Preload("Lessonis").Preload("UserCourses").Create(&course)
	if result.Error != nil {
		http.Error(w, "Не удалось создать курс", http.StatusBadRequest)
		return
	}
	go search.IndexCourse(course, course.ID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(course)
}

func GetAll(w http.ResponseWriter, r *http.Request) {
	var courses []models.Course
	result := initial.DB.Preload("Tests").Preload("Documents").Preload("Lessonis").Preload("UserCourses").Find(&courses)
	if result.Error != nil {
		http.Error(w, "Не удалось получить курсы", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(courses)
}

func GetByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, "ID курса не указан", http.StatusBadRequest)
		return
	}

	var course models.Course
	result := initial.DB.Preload("Tests").Preload("Documents").Preload("Lessonis").Preload("UserCourses").First(&course, id)
	if result.Error != nil {
		http.Error(w, "Курс не найден", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(course)
}

func Update(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Пользователь не авторизован", http.StatusUnauthorized)
		return
	}

	if user.Role != string(middleware.Admin) && user.Role != string(middleware.Teacher) {
		http.Error(w, "Доступ запрещен", http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, "ID курса не указан", http.StatusBadRequest)
		return
	}
	var update models.Course
	err := json.NewDecoder(r.Body).Decode(&update)
	if err != nil {
		http.Error(w, "Ошибка с декодированием", http.StatusBadRequest)
		return
	}

	var course models.Course
	result := initial.DB.First(&course, id)
	if result.Error != nil {
		http.Error(w, "Курс не найден", http.StatusNotFound)
		return
	}
	if update.Name != course.Name {
		update.Name = course.Name
	}
	if update.Description != course.Description || update.Description != "" {
		update.Description = course.Description
	}
	if len(update.Lessonis) != len(course.Lessonis) {
		update.Lessonis = course.Lessonis
	}
	result = initial.DB.Save(&course)
	if result.Error != nil {
		http.Error(w, "Не удалось обновить курс", http.StatusInternalServerError)
		return
	}
	go search.IndexCourse(course, course.ID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(course)
}

func Delete(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Пользователь не авторизован", http.StatusUnauthorized)
		return
	}

	if user.Role != string(middleware.Admin) {
		http.Error(w, "Доступ запрещен", http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, "ID курса не указан", http.StatusBadRequest)
		return
	}

	result := initial.DB.Delete(&models.Course{}, id)
	if result.Error != nil {
		http.Error(w, "Не удалось удалить курс", http.StatusInternalServerError)
		return
	}
	go search.DeleteCourseFromIndex(parseUint(id))
	w.WriteHeader(http.StatusOK)
}

func GetUserCourses(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]
	if userID == "" {
		http.Error(w, "ID пользователя не указан", http.StatusBadRequest)
		return
	}

	var userCourses []models.UserCourse
	result := initial.DB.Where("user_id = ?", userID).Find(&userCourses)
	if result.Error != nil {
		http.Error(w, "Не удалось получить курсы пользователя", http.StatusInternalServerError)
		return
	}

	var courses []models.Course
	for _, uc := range userCourses {
		var course models.Course
		initial.DB.First(&course, uc.CourseID)
		courses = append(courses, course)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(courses)
}

func GetCourseDetails(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, "ID курса не указан", http.StatusBadRequest)
		return
	}

	var course models.Course
	result := initial.DB.Preload("Tests").Preload("Documents").Preload("Lessonis").Preload("UserCourses").First(&course, id)
	if result.Error != nil {
		http.Error(w, "Курс не найден", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(course)
}

func UpdateCourseStatus(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Пользователь не авторизован", http.StatusUnauthorized)
		return
	}

	if user.Role != string(middleware.Admin) && user.Role != string(middleware.Teacher) {
		http.Error(w, "Доступ запрещен", http.StatusForbidden)
		return
	}

	var update models.UserCourse
	err := json.NewDecoder(r.Body).Decode(&update)
	if err != nil {
		http.Error(w, "Ошибка с декодированием", http.StatusBadRequest)
		return
	}

	var userCourse models.UserCourse
	result := initial.DB.Where("user_id = ? AND course_id = ?", update.UserID, update.CourseID).First(&userCourse)
	if result.Error != nil {
		http.Error(w, "Запись о курсе не найдена", http.StatusNotFound)
		return
	}

	if update.Status != userCourse.Status {
		update.Status = userCourse.Status
	}

	result = initial.DB.Save(&userCourse)
	if result.Error != nil {
		http.Error(w, "Не удалось обновить статус", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userCourse)
}

func parseUint(s string) uint {
	u, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return uint(u)
}
