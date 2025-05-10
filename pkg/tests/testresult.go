package tests

import (
	"encoding/json"
	"github.com/segmentio/kafka-go"
	"lms-go/pkg/initial"
	"lms-go/pkg/kfka"
	"lms-go/pkg/middleware"
	"lms-go/pkg/models"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type Answer struct {
	QuestionID uint   `json:"question_id"`
	Answer     string `json:"answer"`
}

type TestResultRequest struct {
	Answers []Answer `json:"answers"`
}

type TestResultDetail struct {
	QuestionID  uint   `json:"question_id"`
	Correct     bool   `json:"correct"`
	UserAnswer  string `json:"user_answer"`
	RightAnswer string `json:"right_answer"`
}

type TestResultResponse struct {
	Score       int                `json:"score"`
	Total       int                `json:"total"`
	Details     []TestResultDetail `json:"details"`
	Status      string             `json:"status"`
	Attempt     int                `json:"attempt"`
	StartedAt   time.Time          `json:"started_at"`
	CompletedAt time.Time          `json:"completed_at"`
}

type StartTestResponse struct {
	Attempt   int       `json:"attempt"`
	StartedAt time.Time `json:"started_at"`
	Duration  int       `json:"duration"`
	Status    string    `json:"status"`
}

func SubmitTest(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}

	var req TestResultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Ошибка декодирования", http.StatusBadRequest)
		return
	}

	vars := mux.Vars(r)
	testID, _ := strconv.Atoi(vars["testID"])

	var test models.Test
	result := initial.DB.First(&test, testID)
	if result.Error != nil {
		http.Error(w, "Тест не найден", http.StatusNotFound)
		return
	}

	var prevResults []models.TestResult
	initial.DB.Where("user_id = ? AND test_id = ?", user.ID, testID).Order("attempt desc").Find(&prevResults)
	attempt := 1
	if len(prevResults) > 0 {
		attempt = prevResults[0].Attempt
	}

	var questions []models.TestQuestion
	initial.DB.Where("test_id = ?", testID).Find(&questions)

	correctAnswers := make(map[uint]string)
	for _, q := range questions {
		correctAnswers[q.ID] = q.CorrectAnswer
	}

	score := 0
	details := make([]TestResultDetail, 0, len(req.Answers))
	for _, ans := range req.Answers {
		correct, ok := correctAnswers[ans.QuestionID]
		isCorrect := ok && (ans.Answer == correct)
		if isCorrect {
			score++
		}
		details = append(details, TestResultDetail{
			QuestionID:  ans.QuestionID,
			Correct:     isCorrect,
			UserAnswer:  ans.Answer,
			RightAnswer: correct,
		})
	}

	status := "не пройден"
	if score >= test.PassMark {
		status = "пройден"
	}

	var testResult models.TestResult
	err := initial.DB.Where("user_id = ? AND test_id = ? AND attempt = ? AND completed_at IS NULL", user.ID, testID, attempt).First(&testResult).Error
	if err != nil {
		http.Error(w, "Нет активной попытки для завершения", http.StatusBadRequest)
		return
	}

	now := time.Now()
	allowedUntil := testResult.StartedAt.Add(time.Duration(test.Duration) * time.Minute)
	if now.After(allowedUntil) {
		testResult.Status = "время вышло"
		testResult.CompletedAt = now
		initial.DB.Save(&testResult)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TestResultResponse{
			Score:       0,
			Total:       len(questions),
			Details:     nil,
			Status:      "время вышло",
			Attempt:     attempt,
			StartedAt:   testResult.StartedAt,
			CompletedAt: now,
		})
		return
	}

	testResult.Score = score
	testResult.Status = status
	testResult.CompletedAt = now
	initial.DB.Save(&testResult)

	resp := TestResultResponse{
		Score:       score,
		Total:       len(questions),
		Details:     details,
		Status:      status,
		Attempt:     attempt,
		StartedAt:   testResult.StartedAt,
		CompletedAt: now,
	}
	/*
		type NotificationTestResult struct {
			TestID    uint   `json:"test_id"`
			UserID    uint   `json:"user_id"`
			TestResID uint   `json:"test_res_id"`
			CourseID  uint   `json:"course_id"`
			LessonID  uint   `json:"lesson_id"`
			Email     string `json:"email"`
			Test      string `json:"test"`
			Course    string `json:"course"`
			Lesson    string `json:"lesson"`
			Score     int    `json:"score"`
			Status    string `json:"status"`
			Event     string `json:"event"`
		}
	*/
	kafkawriter := &kafka.Writer{
		Addr:     kafka.TCP(os.Getenv("KAFKA_ADDRESS")),
		Topic:    "testresult_notifications",
		Balancer: &kafka.LeastBytes{},
	}
	defer kafkawriter.Close()
	var course models.Course
	initial.DB.Where("test_id = ?", testID).First(&course)
	var lesson models.Lessons
	initial.DB.Where("course_id = ?", course.ID).First(&lesson)
	event := kfka.NotificationTestResult{
		TestID:    test.ID,
		UserID:    user.ID,
		TestResID: testResult.ID,
		CourseID:  course.ID,
		LessonID:  lesson.ID,
		Email:     user.Email,
		Test:      test.Title,
		Course:    course.Name,
		Lesson:    lesson.Title,
		Score:     score,
		Status:    status,
	}
	err = event.SendNotRes(kafkawriter)
	if err != nil {
		log.Println("Не удалось отправить сообщение Kafka")
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func StartTest(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	testID, _ := strconv.Atoi(vars["testID"])

	var test models.Test
	result := initial.DB.First(&test, testID)
	if result.Error != nil {
		http.Error(w, "Тест не найден", http.StatusNotFound)
		return
	}

	var prevResults []models.TestResult
	initial.DB.Where("user_id = ? AND test_id = ?", user.ID, testID).Order("attempt desc").Find(&prevResults)
	attempt := 1
	if len(prevResults) > 0 {
		if prevResults[0].CompletedAt.IsZero() {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(StartTestResponse{
				Attempt:   prevResults[0].Attempt,
				StartedAt: prevResults[0].StartedAt,
				Duration:  test.Duration,
				Status:    "уже начата попытка",
			})
			return
		}
		attempt = prevResults[0].Attempt + 1
	}
	if test.MaxAttempts > 0 && attempt > test.MaxAttempts {
		http.Error(w, "Превышено число попыток", http.StatusForbidden)
		return
	}

	startedAt := time.Now()
	res := models.TestResult{
		UserID:    user.ID,
		TestID:    uint(testID),
		Attempt:   attempt,
		StartedAt: startedAt,
	}
	initial.DB.Create(&res)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(StartTestResponse{
		Attempt:   attempt,
		StartedAt: startedAt,
		Duration:  test.Duration,
		Status:    "ok",
	})
}
