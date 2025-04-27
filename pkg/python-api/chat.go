package python_api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"lms-go/pkg/middleware"
	"net/http"
	"os"
)

type ReqChat struct {
	Question string `json:"question"`
}

type RespChat struct {
	Chat_id string `json:"chat_id"`
	Answer  any    `json:"answer"`
}

func AnswerChat(text string) (*RespChat, error) {
	reqBody, err := json.Marshal(ReqChat{Question: text})
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
	_, ok := middleware.GetUserFromContext(r)
	if !ok {
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		http.Error(w, "Не удалось декодировать", http.StatusBadRequest)
		return
	}
	answer, err := AnswerChat(q.Question)
	if err != nil {
		http.Error(w, "Не удалось получить вопрос", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(answer)
}
