package kfka

import (
	"context"
	"encoding/json"
	"github.com/segmentio/kafka-go"
	"log"
)

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

func (n *NotificationTestResult) SendNotRes(writer *kafka.Writer) error {
	e, err := json.Marshal(n)
	if err != nil {
		log.Println("Не удалось перевести в json")
	}
	return writer.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte(n.Event),
		Value: e,
	})
}
