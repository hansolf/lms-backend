package goauth

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"lms-go/pkg/email"
	"lms-go/pkg/initial"
	"lms-go/pkg/middleware"
	"lms-go/pkg/models"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type RegRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type CodeReg struct {
	Code string `json:"code"`
}

func ConstructorReg() *RegRequest {
	return &RegRequest{}
}
func ConstructorCode() *CodeReg {
	return &CodeReg{}
}
func (c *RegRequest) SendEmail(w http.ResponseWriter, r *http.Request) {
	err := json.NewDecoder(r.Body).Decode(c)
	if err != nil {
		http.Error(w, "Не удалось декодировать форму", http.StatusBadRequest)
		return
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_URL"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})
	if strings.Contains(c.Password, "|") || strings.Contains(c.Email, "|") {
		http.Error(w, "Пароль содержит запрещенные символы", http.StatusBadRequest)
		return
	}
	gCode := generateC()
	val := c.Password + "|" + c.Email
	err = rdb.Set(context.Background(), gCode, val, time.Minute*10).Err()
	if err != nil {
		http.Error(w, "Не удалось установить ключи REDIS", http.StatusBadRequest)
		return
	}
	emailD := email.EmailData{
		Code: gCode,
	}
	htmlS, err := emailD.GenerateEmailHTML("EmailCode.html")
	if err != nil {
		http.Error(w, "Не удалось перевести в строку", http.StatusBadRequest)
		return
	}
	to := []string{c.Email}
	topic := "Новое уведомление от LMS"
	err = email.SendEmail(to, topic, htmlS)
	if err != nil {
		http.Error(w, "Не удалось отправить письмо", http.StatusBadRequest)
		return
	}
}
func (cr *CodeReg) SignUp(w http.ResponseWriter, r *http.Request) {
	err := json.NewDecoder(r.Body).Decode(cr)
	if err != nil {
		http.Error(w, "Ошибка с декодированием", http.StatusBadRequest)
		return
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_URL"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})
	val, err := rdb.Get(context.Background(), cr.Code).Result()
	if err != nil {
		http.Error(w, "Неправильный код", http.StatusBadRequest)
		return
	}
	stringPE := strings.Split(val, "|")
	// хэшируем пароль
	hash, err := bcrypt.GenerateFromPassword([]byte(stringPE[0]), 10)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusBadRequest)
		return
	}
	//добавляем юзера в бд
	user := models.User{
		Email:    stringPE[1],
		Password: string(hash),
	}
	result := initial.DB.Create(&user)
	if result.Error != nil {
		http.Error(w, "Не удалось создать пользователя", http.StatusBadRequest)
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID,
		"exp": time.Now().Add(time.Hour * 72).Unix(),
	})
	tokenstring, err := token.SignedString([]byte(os.Getenv("SECRET")))
	if err != nil {
		http.Error(w, "не удалось создать токен", http.StatusNotFound)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    tokenstring,
		Path:     "/",
		Expires:  time.Now().Add(time.Hour * 72),
		Secure:   false,
		HttpOnly: true,
	})
	// успех
	resp := struct {
		Data models.User `json:"data"`
	}{
		Data: user,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func Login(w http.ResponseWriter, r *http.Request) {
	//также получаем п,л,п
	var userR models.User
	err := json.NewDecoder(r.Body).Decode(&userR)
	if err != nil {
		http.Error(w, "Ошибка с декодированием", http.StatusBadRequest)
		return
	}
	// найти пользователя
	var user models.User
	initial.DB.First(&user, "email = ?", userR.Email)
	if user.ID == 0 {
		http.Error(w, "неверная почта или пароль", http.StatusNotFound)
		return
	}
	//сравниваю хэщ и пароль
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(userR.Password))
	if err != nil {
		http.Error(w, "неверный пароль", http.StatusUnauthorized)
		return
	}
	// генерим jwt токен
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.ID,
		"exp": time.Now().Add(time.Hour * 72).Unix(),
	})
	tokenstring, err := token.SignedString([]byte(os.Getenv("SECRET")))
	if err != nil {
		http.Error(w, "не удалось создать токен", http.StatusNotFound)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    tokenstring,
		Path:     "/",
		Expires:  time.Now().Add(time.Hour * 72),
		Secure:   false,
		HttpOnly: true,
	})
	response := struct {
		Data models.User `json:"data"`
	}{
		Data: user,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func UpdateMe(w http.ResponseWriter, r *http.Request) {
	userR, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}

	type UserUpdateRequest struct {
		Name       string `json:"name"`
		Secondname string `json:"secondname"`
		Vuz        string `json:"vuz"`
		Kafedra    string `json:"kafedra"`
		Fakultet   string `json:"fakultet"`
	}

	var update UserUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Не удалось задекодировать", http.StatusBadRequest)
		return
	}

	var user models.User
	if err := initial.DB.First(&user, "id = ?", userR.ID).Error; err != nil {
		http.Error(w, "Не удалось найти пользователя", http.StatusNotFound)
		return
	}

	changed := false
	if update.Name != "" && update.Name != user.Name {
		user.Name = update.Name
		changed = true
	}
	if update.Secondname != "" && update.Secondname != user.Secondname {
		user.Secondname = update.Secondname
		changed = true
	}
	if update.Vuz != "" && update.Vuz != user.Vuz {
		user.Vuz = update.Vuz
		changed = true
	}
	if update.Kafedra != "" && update.Kafedra != user.Kafedra {
		user.Kafedra = update.Kafedra
		changed = true
	}
	if update.Fakultet != "" && update.Fakultet != user.Fakultet {
		user.Fakultet = update.Fakultet
		changed = true
	}

	if changed {
		if err := initial.DB.Save(&user).Error; err != nil {
			http.Error(w, "Ошибка при сохранении", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		Expires:  time.Now().Add(-time.Hour),
		MaxAge:   -1,
		Secure:   false,
		HttpOnly: true,
	})
}

func Me(w http.ResponseWriter, r *http.Request) {
	userR, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}
	if userR.Role != string(middleware.Admin) && userR.Role != string(middleware.Teacher) {
		var user models.User
		initial.DB.First(&user, "id = ?", userR.ID)
		if user.ID == 0 {
			http.Error(w, "Пользователь не найден", http.StatusNotFound)
		}
		response := struct {
			ID         uint                `json:"id"`
			Name       string              `json:"name"`
			Secondname string              `json:"secondname"`
			Fakultet   string              `json:"kafedra"`
			Vuz        string              `json:"vuz"`
			MyCourses  []models.UserCourse `json:"myCourses"`
			Email      string              `json:"email"`
			Role       string              `json:"role"`
			CreatedAt  time.Time           `json:"created_at"`
		}{
			ID:         user.ID,
			Name:       user.Name,
			Secondname: user.Secondname,
			Fakultet:   user.Fakultet,
			Vuz:        user.Vuz,
			MyCourses:  user.UserCourses,
			Email:      user.Email,
			Role:       user.Role,
			CreatedAt:  user.CreatedAt,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	} else {
		var user models.User
		initial.DB.First(&user, "id = ?", userR.ID)
		if user.ID == 0 {
			http.Error(w, "Пользователь не найден", http.StatusNotFound)
		}
		response := struct {
			ID         uint                `json:"id"`
			Name       string              `json:"name"`
			Secondname string              `json:"secondname"`
			Kafedra    string              `json:"kafedra"`
			Vuz        string              `json:"vuz"`
			MyCourses  []models.UserCourse `json:"myCourses"`
			Email      string              `json:"email"`
			Role       string              `json:"role"`
			CreatedAt  time.Time           `json:"created_at"`
		}{
			ID:         user.ID,
			Name:       user.Name,
			Secondname: user.Secondname,
			Kafedra:    user.Kafedra,
			Vuz:        user.Vuz,
			MyCourses:  user.UserCourses,
			Email:      user.Email,
			Role:       user.Role,
			CreatedAt:  user.CreatedAt,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

type ReqToVerify struct {
	Name       string `json:"name"`
	Secondname string `json:"secondname"`
	Vuz        string `json:"vuz"`
	Kafedra    string `json:"kafedra"`
}

func Constructor() *ReqToVerify {
	return &ReqToVerify{}
}

func (v *ReqToVerify) SendVerTeach(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}
	if user.Role == string(middleware.Teacher) {
		http.Error(w, "Вы уже учитель", http.StatusNotFound)
		return
	}
	err := json.NewDecoder(r.Body).Decode(&v)
	if err != nil {
		http.Error(w, "Не удалось декодировать", http.StatusBadRequest)
		return
	}
	if strings.Contains(v.Name, "|") && strings.Contains(user.Email, "|") {
		http.Error(w, "Форма содержит запрещенные символы", http.StatusBadRequest)
		return
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_URL"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})
	val := user.Email + "|" + v.Name + "|" + v.Secondname + "|" + v.Vuz + "|" + v.Kafedra
	strID := strconv.Itoa(int(user.ID))
	err = rdb.Set(context.Background(), strID, val, 7*60*24*time.Minute).Err()
	if err != nil {
		http.Error(w, "Не удалось установить ключи REDIS", http.StatusBadRequest)
		return
	}
	//'СПбГЭУ', 'НИУ ИТМО', 'СПбПУ Петра Великого'
	emailD := email.EmailData{
		NameTeacher:       v.Name,
		SecondNameTeacher: v.Secondname,
		Vuz:               v.Vuz,
		Kafedra:           v.Kafedra,
		VerifyLink:        "http://localhost:8080/api/verifyteach/" + strID, // добавить
	}
	if v.Vuz == "СПбГЭУ" {
		var htmlS string
		htmlS, err = emailD.GenerateEmailHTML("VerifyTeacher.html")
		if err != nil {
			http.Error(w, "Не удалось перевести html страницу в строку", http.StatusBadRequest)
			return
		}
		to := []string{os.Getenv("UNECON_ADMIN")}
		subject := "Уведомление от LMS"
		err = email.SendEmail(to, subject, htmlS)
		if err != nil {
			http.Error(w, "Не удалось отправить на почту админа", http.StatusBadRequest)
			return
		}
	}

	if v.Vuz == "СПбПУ Петра Великого" {
		var htmlS string
		htmlS, err = emailD.GenerateEmailHTML("VerifyTeacher.html")
		if err != nil {
			http.Error(w, "Не удалось перевести html страницу в строку", http.StatusBadRequest)
			return
		}
		to := []string{os.Getenv("SPBPU_ADMIN")}
		subject := "Уведомление от LMS"
		err = email.SendEmail(to, subject, htmlS)
		if err != nil {
			http.Error(w, "Не удалось отправить на почту админа", http.StatusBadRequest)
			return
		}
	}

	if v.Vuz == "НИУ ИТМО" {
		var htmlS string
		htmlS, err = emailD.GenerateEmailHTML("VerifyTeacher.html")
		if err != nil {
			http.Error(w, "Не удалось перевести html страницу в строку", http.StatusBadRequest)
			return
		}
		to := []string{os.Getenv("ITMO_ADMIN")}
		subject := "Уведомление от LMS"
		err = email.SendEmail(to, subject, htmlS)
		if err != nil {
			http.Error(w, "Не удалось отправить на почту админа", http.StatusBadRequest)
			return
		}
	}
}

func VerifyTeacher(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["id"]
	var user models.User
	initial.DB.First(&user, userID)
	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_URL"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})
	val, err := rdb.Get(context.Background(), userID).Result()
	if err != nil {
		http.Error(w, "Не удалось получить данные REDIS", http.StatusBadRequest)
		return
	}
	data := strings.Split(val, "|")
	user.Role = "преподаватель"
	user.Email = data[0]
	user.Name = data[1]
	user.Secondname = data[2]
	user.Vuz = data[3]
	user.Kafedra = data[4]
	if err := initial.DB.Save(&user).Error; err != nil {
		http.Error(w, "Не удалось обновить пользователя", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
func generateC() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}
