package goauth

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"lms-go/pkg/email"
	"lms-go/pkg/initial"
	"lms-go/pkg/models"
	"math/rand"
	"net/http"
	"os"
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
	gCode := generateC()
	value := c.Password + "|" + c.Email
	err = rdb.Set(context.Background(), gCode, value, time.Minute*10).Err()
	if err != nil {
		http.Error(w, "Не удалось установить", http.StatusBadRequest)
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
	s := strings.Split(val, "|")
	// хэшируем пароль
	hash, err := bcrypt.GenerateFromPassword([]byte(s[0]), 10)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusBadRequest)
		return
	}
	//добавляем юзера в бд
	user := models.User{
		Email:    s[1],
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
	initial.DB.First(&user, "email = ?", user.Email)
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

func Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour * 72),
		Secure:   false,
		HttpOnly: true,
	})
	w.WriteHeader(http.StatusOK)
}

func Me(w http.ResponseWriter, r *http.Request) {
	cookie, _ := r.Cookie("token")
	token, _ := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("SECRET")), nil
	})
	claims := token.Claims.(jwt.MapClaims)
	userID := uint(claims["sub"].(float64))
	var user models.User
	initial.DB.First(&user, "id = ?", userID)
	if user.ID == 0 {
		http.Error(w, "Пользователь не найден", http.StatusNotFound)
	}
	eToken := cookie.Value
	if eToken == "" {
		http.Error(w, `{"error": "Неверный токен"}`, http.StatusUnauthorized)
		return
	}
	response := struct {
		ID        uint      `json:"id"`
		Name      string    `json:"name"`
		Email     string    `json:"email"`
		Role      string    `json:"role"`
		CreatedAt time.Time `json:"created_at"`
	}{
		ID:        user.ID,
		Name:      user.Username,
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
	}
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
	}
}

func generateC() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}
