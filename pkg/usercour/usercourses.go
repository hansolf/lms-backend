package usercour

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"lms-go/pkg/initial"
	"lms-go/pkg/middleware"
	"lms-go/pkg/models"
	"net/http"
	"strconv"
)

func EnrollInCourse(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Пользователь не авторизован", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	courseID := vars["courseID"]
	if courseID == "" {
		http.Error(w, "ID курса не указан", http.StatusBadRequest)
		return
	}

	var existing models.UserCourse
	result := initial.DB.Where("user_id = ? AND course_id = ?", user.ID, courseID).First(&existing)
	if result.Error == nil {
		http.Error(w, "Вы уже записаны на этот курс", http.StatusBadRequest)
		return
	}

	userCourse := models.UserCourse{
		UserID:   user.ID,
		CourseID: parseUint(courseID),
	}
	if err := initial.DB.Create(&userCourse).Error; err != nil {
		http.Error(w, "Не удалось записаться на курс", http.StatusInternalServerError)
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
