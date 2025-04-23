package middleware

import (
	"context"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"lms-go/pkg/initial"
	"lms-go/pkg/models"
	"net/http"
	"os"
)

type Role string

const (
	Admin   Role = "администратор"
	Teacher Role = "преподаватель"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("token")
		if err != nil {
			http.Error(w, "Не авторизован", http.StatusUnauthorized)
			return
		}
		token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(os.Getenv("SECRET")), nil
		})
		claims := token.Claims.(jwt.MapClaims)
		userID := claims["sub"].(float64)
		var user models.User
		result := initial.DB.First(&user, userID)
		if result.Error != nil {
			http.Error(w, "Пользователь не найден", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), "user", user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetUserFromContext(r *http.Request) (models.User, bool) {
	user, ok := r.Context().Value("user").(models.User)
	return user, ok
}
