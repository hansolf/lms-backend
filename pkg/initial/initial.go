package initial

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"
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
		log.Fatal("Ошибка загрузки .env")
	}
}
func ConRedis() {
	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_URL"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})
	err := rdb.Set(context.Background(), "key", "value", 0).Err()
	if err != nil {
		log.Fatal("Не удалось установить ключ REDIS ", err)
	}
	val, err := rdb.Get(context.Background(), "key").Result()
	if err != nil {
		log.Fatal("Не удалось получить значение по ключу")
	}
	fmt.Println("key", val)
}
func ConDB() {
	var err error
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"))
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
	Client, err = minio.New("localhost:9000", &minio.Options{
		Creds:  credentials.NewStaticV4("minioadmin", "minioadmin", ""),
		Secure: false,
	})
	if err != nil {
		panic("Не удалось подключиться к Minio")
	}
}
