package kfka

import (
	"context"
	"encoding/json"
	"github.com/segmentio/kafka-go"
	"log"
)

type NotificationTest struct {
	UserID   uint   `json:"user_id"`
	CourseID uint   `json:"course_id"`
	LessonID uint   `json:"lesson_id"`
	Email    string `json:"email"`
	Test     string `json:"test"`
	Course   string `json:"course"`
	Lesson   string `json:"lesson"`
	Event    string `json:"event"`
}

func (n *NotificationTest) SendNotTest(writer *kafka.Writer) error {
	e, err := json.Marshal(n)
	if err != nil {
		log.Println("Ошибка перевод в json ", err)
	}
	return writer.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte(n.Event),
		Value: e,
	})
}
