package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/segmentio/kafka-go"
	"lms-go/pkg/email"
	"lms-go/pkg/initial"
	"lms-go/pkg/kfka"
	"lms-go/pkg/middleware"
	"lms-go/pkg/routes"
	"log"
	"net/http"
)

func init() {
	initial.LoadEnvComp()
	initial.ConDB()
	initial.SyncDB()
	initial.InitES()
	initial.InitEScs()
	initial.InitESls()
	initial.InitMinio()
}

func main() {
	r := mux.NewRouter()
	routes.SetupAuth(r)
	meRouter := r.PathPrefix("/api").Subrouter()
	meRouter.Use(middleware.AuthMiddleware)
	routes.SetupMe(meRouter)
	chatRouter := r.PathPrefix("/api").Subrouter()
	chatRouter.Use(middleware.AuthMiddleware)
	routes.SetupChat(chatRouter)
	coursesRouter := r.PathPrefix("/api/courses").Subrouter()
	coursesRouter.Use(middleware.AuthMiddleware)
	routes.SetupCourses(coursesRouter)
	lessonRouter := r.PathPrefix("/api/courses/{courseID}").Subrouter()
	lessonRouter.Use(middleware.AuthMiddleware)
	routes.SetupLessons(lessonRouter)
	testsRouter := r.PathPrefix("/api/courses/{courseID}/lessons/{lessonID}").Subrouter()
	testsRouter.Use(middleware.AuthMiddleware)
	routes.SetupTests(testsRouter)
	vidRouter := r.PathPrefix("/api/courses/{courseID}/lessons/{lessonID}").Subrouter()
	vidRouter.Use(middleware.AuthMiddleware)
	routes.SetupVideos(vidRouter)
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5176"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})
	go func() {
		kafkaReader := kafka.NewReader(kafka.ReaderConfig{
			Brokers: []string{"localhost:9092"},
			Topic:   "test_notifications",
			GroupID: "test-notifications-consumer-group",
		})
		defer kafkaReader.Close()
		for {
			m, err := kafkaReader.ReadMessage(context.Background())
			if err != nil {
				log.Println("Не удалось прочитать сообщение Kafka: ", err)
			}
			var event kfka.NotificationTest
			err = json.Unmarshal(m.Value, &event)
			if err != nil {
				log.Println("Не удалось перевести json в структуру: ", err)
			}
			var html email.EmailData
			htmlq, err := html.GenerateEmailHTML("NewTest.html")
			if err != nil {
				log.Println("Не удалось перевести в строку ", err)
			}
			to := []string{event.Email}
			object := "Новое сообщение от курса" + event.Course
			err = email.SendEmail(to, object, htmlq)
			if err != nil {
				log.Println("Не удалось отправить на почту ", err)
			}
		}
	}()
	go func() {
		kafkaReader := kafka.NewReader(kafka.ReaderConfig{
			Brokers: []string{"localhost:9092"},
			Topic:   "course_update_notifications",
			GroupID: "notifications-update-consumer-group",
		})
		defer kafkaReader.Close()
		for {
			m, err := kafkaReader.ReadMessage(context.Background())
			if err != nil {
				log.Println("Не удалось прочитать сообщение Kafka", err)
			}
			var notification kfka.NotificationCourse
			err = json.Unmarshal(m.Value, &notification)
			if err != nil {
				log.Println("Не удалость преобразовать в структуру", err)
			}
			emailD := email.EmailData{
				CourseName:        notification.CourseName,
				LessonTitle:       notification.Lesson,
				LessonDescription: notification.Description,
				LessonLink:        fmt.Sprintf("http://localhost:8080/api/courses/%v/lessons/%v", notification.CourseID, notification.LessonId),
			}
			gHtml, err := emailD.GenerateEmailHTML("UpdateLesson.html")
			if err != nil {
				log.Println("Ошибка в генерации html ", err)
			}
			to := []string{notification.Email}
			object := "Новое сообщение от курса " + notification.CourseName
			err = email.SendEmail(to, object, gHtml)
			if err != nil {
				log.Println("Не удалось отправить письмо ", err)
			}
		}
	}()
	go func() {
		kafkaReader := kafka.NewReader(kafka.ReaderConfig{
			Brokers: []string{"localhost:9092"},
			Topic:   "course_notifications",
			GroupID: "notifications-consumer-group",
		})
		defer kafkaReader.Close()
		for {
			m, err := kafkaReader.ReadMessage(context.Background())
			if err != nil {
				log.Printf("Ошибка прочтения сообщения Kafka: %s\n", err)
				continue
			}
			var event kfka.NotificationCourse
			err = json.Unmarshal(m.Value, &event)
			if err != nil {
				log.Printf("Ошибка json -> struct: %s\n", err)
				continue
			}
			emailD := email.EmailData{
				CourseName:        event.CourseName,
				LessonTitle:       event.Lesson,
				LessonDescription: event.Description,
				LessonLink:        fmt.Sprintf("http://localhost:8080/api/courses/%s/lessons/%s", event.CourseID, event.LessonId),
			}
			htmlBody, err := emailD.GenerateEmailHTML("NewLesson.html")
			if err != nil {
				log.Println("Ошибка генерации письма:", err)
				continue
			}
			to := []string{event.Email}
			subject := "Новое сообщение от курса " + emailD.CourseName
			if err = email.SendEmail(to, subject, htmlBody); err != nil {
				log.Println("Ошибка отправки email:", err)
			}
		}
	}()
	handler := c.Handler(r)
	log.Printf("Сервер запущен на http://localhost:8080")
	err := http.ListenAndServe(":8080", handler)
	if err != nil {
		log.Fatal("Ошибка запуска сервера:", err)
	}
}
