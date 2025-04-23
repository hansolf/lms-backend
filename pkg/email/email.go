package email

import (
	"bytes"
	"html/template"
	"net/smtp"
	"os"
)

type EmailRequest struct {
	ToAddress string `json:"to_addr"`
	Subject   string `json:"subject"`
	Body      string `json:"body"`
}

type EmailData struct {
	CourseName        string `json:"course_name"`
	LessonTitle       string `json:"lesson_title"`
	LessonDescription string `json:"lesson_description"`
	TestTitle         string `json:"test_title"`
	LessonLink        string `json:"lesson_link"`
	TestLink          string `json:"test_link"`
	Code              string `json:"code"`
}

func SendEmail(to []string, subject, html string) error {
	auth := smtp.PlainAuth("", os.Getenv("EMAIL"), os.Getenv("EMAILPASS"), os.Getenv("SMTP"))
	headers := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";"
	message := "Subject: " + subject + "\n" + headers + "\n\n" + html
	return smtp.SendMail(os.Getenv("SMTP_ADDR"), auth, os.Getenv("EMAIL"), to, []byte(message))
}

func (data *EmailData) GenerateEmailHTML(html string) (string, error) {
	tmpl, err := template.New("Email").ParseFiles("C:/Users/vitek the g/GolandProjects/lms-go/templateshtml/" + html)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
