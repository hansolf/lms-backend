package initial

import (
	"crypto/tls"
	"fmt"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"lms-go/pkg/models"
	"lms-go/pkg/search"
	"log"
	"net/http"
	"os"
)

var DB *gorm.DB
var Client *minio.Client

func LoadEnvComp() {
	err := godotenv.Load()
	if err != nil {
		log.Println(".env файл не найден, используются переменные окружения из системы")
	}
}
func ConDB() {
	var (
		dbHost     = os.Getenv("DB_HOST")
		dbUser     = os.Getenv("DB_USER")
		dbPassword = os.Getenv("DB_PASSWORD")
		dbName     = os.Getenv("DB_NAME")
		dbPort     = os.Getenv("DB_PORT")
	)
	fmt.Printf("DB Config: Host=%s, User=%s, DBName=%s, Port=%s\n",
		dbHost, dbUser, dbName, dbPort)

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		dbHost, dbUser, dbPassword, dbName, dbPort)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("дб не подключается")
	}
	err = DB.Exec(`
		DO $$ BEGIN
			CREATE TYPE user_role AS ENUM ('студент', 'преподаватель', 'администратор');
		EXCEPTION
			WHEN duplicate_object THEN null;
		END $$;

		DO $$ BEGIN
			CREATE TYPE user_course_status AS ENUM ('не пройден', 'в процессе', 'пройден');
		EXCEPTION
			WHEN duplicate_object THEN null;
		END $$;

		DO $$ BEGIN
			CREATE TYPE question_types AS ENUM ('одиночный выбор', 'множественный выбор', 'текстовый ответ');
		EXCEPTION
			WHEN duplicate_object THEN null;
		END $$;

		DO $$ BEGIN
			CREATE TYPE test_result_status AS ENUM ('не выполнено', 'в процессе', 'выполнено', 'провалено');
		EXCEPTION
			WHEN duplicate_object THEN null;
		END $$;

		DO $$ BEGIN
			CREATE TYPE kafedra AS ENUM ('не указано', 'Кафедра информатики', 'Кафедра философии', 'Кафедра математики');
		EXCEPTION
			WHEN duplicate_object THEN null;
		END $$;

		DO $$ BEGIN
			CREATE TYPE vuz AS ENUM ('не указано','СПбГЭУ', 'НИУ ИТМО', 'СПбПУ Петра Великого');
		EXCEPTION
			WHEN duplicate_object THEN null;
		END $$;

		DO $$ BEGIN
			CREATE TYPE fakultet AS ENUM ('не указано','Факультет информатики', 'Факультет философии', 'Факультет математики');
		EXCEPTION
			WHEN duplicate_object THEN null;
		END $$;

	`).Error
	if err != nil {
		log.Fatalf("Ошибка создания enumов: %v", err)
	}
}

func SyncDB() {
	DB.AutoMigrate(
		&models.User{},
		&models.Course{},
		&models.Lessons{},
		&models.Test{},
		&models.TestQuestion{},
		&models.TestResult{},
		&models.Documenti{},
		&models.UserCourse{},
		&models.TestVideo{},
		&models.ResponseChat{},
		&models.ResponseSummary{},
	)
}

func InitES() {
	cfg := elasticsearch.Config{
		Addresses: []string{os.Getenv("ES")},
		Username:  "elastic",
		Password:  os.Getenv("PASS_ES"),
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		panic("Ошибка подключения к ElasticSearch")
	}
	search.ES = client

}
func InitEScs() {
	var courses []models.Course
	DB.Find(&courses)
	for _, course := range courses {
		search.IndexCourse(course, course.ID)
	}
}
func InitESls() {
	var lessons []models.Lessons
	DB.Find(&lessons)
	for _, lesson := range lessons {
		search.IndexLesson(lesson, lesson.ID)
	}
}

func InitMinio() {
	var err error
	endpoint := os.Getenv("MINIO_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:9000"
	}

	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	if accessKey == "" {
		accessKey = "minioadmin"
	}

	secretKey := os.Getenv("MINIO_SECRET_KEY")
	if secretKey == "" {
		secretKey = "minioadmin"
	}

	Client, err = minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	if err != nil {
		panic("Не удалось подключиться к Minio")
	}
}
