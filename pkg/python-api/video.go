package python_api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/minio/minio-go/v7"
	"io/ioutil"
	"lms-go/pkg/initial"
	"lms-go/pkg/middleware"
	"lms-go/pkg/models"
	"net/http"
	"os"
	"strconv"
)

func UploadVid(w http.ResponseWriter, r *http.Request) {
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
	if handler.Header.Get("Content-Type") != "video/mp4" {
		http.Error(w, "Загружайте только видео", http.StatusBadRequest)
		return
	}
	doc := models.TestVideo{
		CourseID: parseUint(courseID),
		LessonID: parseUint(lessonID),
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
	_, err = initial.Client.PutObject(context.Background(),
		"lectures", handler.Filename,
		file,
		handler.Size, minio.PutObjectOptions{ContentType: handler.Header.Get("Content-Type")})

	if err != nil {
		http.Error(w, "Ошибка загрузки в MinIO", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

func StreamVid(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	docID := vars["vidID"]
	if docID == "" {
		http.Error(w, "Видео не найдено", http.StatusBadRequest)
		return
	}
	var doc models.TestVideo
	result := initial.DB.First(&doc, docID)
	if result.Error != nil {
		http.Error(w, "Видео нет в базе данных", http.StatusInternalServerError)
		return
	}
	obj, err := initial.Client.GetObject(
		context.Background(),
		"lectures",
		doc.FileName,
		minio.GetObjectOptions{},
	)
	if err != nil {
		http.Error(w, "Ошибка при получении видео из MinIO", http.StatusInternalServerError)
		return
	}
	defer obj.Close()
	w.Header().Set("Content-Type", doc.FileType)
	w.Header().Set("Accept-Ranges", "bytes")
	http.ServeContent(w, r, doc.FileName, doc.CreatedAt, obj)
}

type ProcessRequest struct {
	Filename string `json:"filename"`
}

type ProcessResponse struct {
	Filename   string      `json:"filename"`
	Transcript string      `json:"transcript"`
	Summary    interface{} `json:"summary"`
	Timestamps interface{} `json:"timestamps"`
	Cached     bool        `json:"cached"`
}

func GetVideoSummary(filename string) (*ProcessResponse, error) {
	reqBody, err := json.Marshal(ProcessRequest{Filename: filename})
	if err != nil {
		return nil, err
	}

	resp, err := http.Post("http://"+os.Getenv("PYTHON_SUM")+"/process/", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result ProcessResponse
	if err = json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("ошибка парсинга ответа: %w\nraw: %s", err, string(body))
	}

	return &result, nil
}

func VideoSummary(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vidID := vars["vidID"]
	if vidID == "" {
		http.Error(w, "Не указано имя файла", http.StatusBadRequest)
		return
	}
	var doc models.TestVideo
	initial.DB.First(&doc, "id = ?", vidID)
	if doc.ID == 0 {
		http.Error(w, "Видео не найдено", http.StatusBadRequest)
	}
	summary, err := GetVideoSummary(doc.FileName)
	if err != nil {
		http.Error(w, "Ошибка получения summary: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

func parseUint(s string) uint {
	u, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return uint(u)
}
