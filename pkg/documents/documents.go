package documents

import (
	"context"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/minio/minio-go/v7"
	"io"
	"lms-go/pkg/initial"
	"lms-go/pkg/middleware"
	"lms-go/pkg/models"
	"net/http"
	"strconv"
)

func UploadDoc(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}
	if user.Role != string(middleware.Admin) && user.Role != string(middleware.Teacher) {
		http.Error(w, "Доступ запрещен", http.StatusForbidden)
		return
	}
	err := r.ParseMultipartForm(1024 * 10 << 20) // 10 гб
	if err != nil {
		http.Error(w, "Ошибка при разборе формы", http.StatusBadRequest)
		return
	}
	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Ошибка при получении файла", http.StatusBadRequest)
		return
	}
	defer file.Close()

	vars := mux.Vars(r)
	courseID := vars["courseID"]
	if courseID == "" {
		http.Error(w, "ID курса не указан", http.StatusBadRequest)
		return
	}
	vars = mux.Vars(r)
	lessonID := vars["lessonID"]
	if lessonID == "" {
		http.Error(w, "ID урока не указан", http.StatusBadRequest)
		return
	}

	_, err = initial.Client.PutObject(context.Background(),
		"uploadsdoc", handler.Filename,
		file,
		handler.Size, minio.PutObjectOptions{ContentType: handler.Header.Get("Content-Type")})

	if err != nil {
		http.Error(w, "Ошибка загрузки в MinIO", http.StatusInternalServerError)
		return
	}
	courseIDUint := parseUint(courseID)
	lessonIDUint := parseUint(lessonID)
	doc := models.Documenti{
		CourseID: courseIDUint,
		LessonID: lessonIDUint,
		FileName: handler.Filename,
		FileType: handler.Header.Get("Content-Type"),
		FilePath: handler.Filename,
		FileSize: float64(handler.Size),
	}
	result := initial.DB.Create(&doc)
	if result.Error != nil {
		http.Error(w, "ошибка при сохранении информации о файле", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

func DownloadDoc(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	docID := vars["id"]
	if docID == "" {
		http.Error(w, "Документ не найден", http.StatusBadRequest)
		return
	}
	var doc models.Documenti
	result := initial.DB.First(&doc, docID)
	if result.Error != nil {
		http.Error(w, "Документа нет в базе данных", http.StatusInternalServerError)
		return
	}
	obj, err := initial.Client.GetObject(context.Background(), "uploadsdoc", doc.FileName, minio.GetObjectOptions{})
	if err != nil {
		http.Error(w, "Не удалось скачать файл", http.StatusInternalServerError)
		return
	}
	defer obj.Close()
	w.Header().Set("Content-Disposition", "inline; filename=\""+doc.FileName+"\"")
	w.Header().Set("Content-Type", doc.FileType)
	io.Copy(w, obj)
}

func parseUint(s string) uint {
	u, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return uint(u)
}
