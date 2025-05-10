package python_api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"lms-go/pkg/initial"
	"lms-go/pkg/middleware"
	"lms-go/pkg/models"
	"net/http"
	"os"
)

type ReqChat struct {
	Question string `json:"question"`
	UserID   uint   `json:"user_id"`
}

type RespChat struct {
	Chat_id string `json:"chat_id"`
	Answer  any    `json:"answer"`
	UserID  uint   `json:"user_id"`
}

func AnswerChat(text string, userID uint) (*RespChat, error) {
	reqBody, err := json.Marshal(ReqChat{
		Question: text,
		UserID:   userID,
	})
	if err != nil {
		return nil, err
	}
	resp, err := http.Post("http://"+os.Getenv("PYTHON_CHAT")+"/api/chat", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API ошибка: %s", string(body))
	}

	var chatResp RespChat
	err = json.Unmarshal(body, &chatResp)
	if err != nil {
		return nil, err
	}
	return &chatResp, nil
}

func Constructor() *ReqChat {
	return &ReqChat{}
}

func (q *ReqChat) AnswerChatBot(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}
	var chat models.ResponseChat
	err := json.NewDecoder(r.Body).Decode(&chat)
	if err != nil {
		http.Error(w, "Не удалось декодировать", http.StatusBadRequest)
		return
	}
	answer, err := AnswerChat(chat.Answer, user.ID)
	if err != nil {
		http.Error(w, "Не удалось получить вопрос"+err.Error(), http.StatusBadRequest)
		return
	}
	chat.UserID = user.ID
	chat.Response = answer
	initial.DB.Create(&chat)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(answer)
}

func GetMyChats(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}
	var chats []models.ResponseChat
	err := initial.DB.Where("user_id = ?", user.ID).Find(&chats)
	if err != nil {
		http.Error(w, "Не удалось найти чаты", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chats)
}
