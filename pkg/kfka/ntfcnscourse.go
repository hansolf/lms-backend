package kfka

import (
	"context"
	"encoding/json"
	"github.com/segmentio/kafka-go"
)

type NotificationCourse struct {
	UserID      uint   `json:"user_id"`
	CourseID    uint   `json:"course_id"`
	LessonId    uint   `json:"lesson_id"`
	CourseName  string `json:"course_name"`
	Email       string `json:"email"`
	Lesson      string `json:"lesson"`
	Description string `json:"description"`
	EventType   string `json:"event_type"`
	Text        string `json:"text"`
}

func (e *NotificationCourse) SendNotif(writer *kafka.Writer) error {
	msg, err := json.Marshal(e)
	if err != nil {
		return err
	}
	return writer.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte(e.EventType),
		Value: msg,
	})
}
