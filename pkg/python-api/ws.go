package python_api

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Message struct {
	Type      string    `json:"type"`
	UserID    int       `json:"user_id"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

func closeConnection(conn *websocket.Conn) {
	conn.WriteControl(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		time.Now().Add(time.Second))
	conn.Close()
}

func ChatWebSocket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userid := vars["id"]
	userID, _ := strconv.Atoi(userid)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Ошибка обновления до WebSocket: %v", err)
		return
	}
	defer closeConnection(conn)

	// Подключаемся к Python-сервису
	pythonConn, err := connectToPythonService()
	if err != nil {
		log.Printf("Ошибка подключения к Python-сервису: %v", err)
		return
	}
	defer closeConnection(pythonConn)

	// Контекст для отмены операций
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Каналы для сообщений
	fromClient := make(chan Message)
	fromPython := make(chan Message)
	var wg sync.WaitGroup

	// Настраиваем таймауты
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	pythonConn.SetReadDeadline(time.Now().Add(60 * time.Second))
	pythonConn.SetPongHandler(func(string) error {
		pythonConn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Горутина для пинг-понга с клиентом
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
					log.Printf("Ошибка пинга клиенту: %v", err)
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Горутина для пинг-понга с Python
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := pythonConn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
					log.Printf("Ошибка пинга Python: %v", err)
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Горутина для чтения сообщений от клиента
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(fromClient)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				var msg Message
				err := conn.ReadJSON(&msg)
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						log.Printf("Ошибка чтения от клиента: %v", err)
					}
					cancel() // Отменяем контекст при ошибке
					return
				}

				msg.UserID = userID
				msg.Timestamp = time.Now()
				fromClient <- msg
			}
		}
	}()

	// Горутина для чтения сообщений от Python-сервиса
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(fromPython)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				var msg Message
				err := pythonConn.ReadJSON(&msg)
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						log.Printf("Ошибка чтения от Python: %v", err)
					}
					cancel() // Отменяем контекст при ошибке
					return
				}
				fromPython <- msg
			}
		}
	}()

	for {
		select {
		case msg, ok := <-fromClient:
			if !ok {
				cancel()
				return
			}

			if err := pythonConn.WriteJSON(msg); err != nil {
				log.Printf("Ошибка отправки в Python: %v", err)
				cancel()
				return
			}

		case msg, ok := <-fromPython:
			if !ok {
				cancel()
				return
			}

			if err := conn.WriteJSON(msg); err != nil {
				log.Printf("Ошибка отправки клиенту: %v", err)
				cancel()
				return
			}

		case <-ctx.Done():
			return
		}
	}
}

func connectToPythonService() (*websocket.Conn, error) {
	pythonURL := "ws://" + os.Getenv("PYTHON_CHAT") + "/ws/chat"
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}
	header := http.Header{}

	conn, resp, err := dialer.Dial(pythonURL, header)
	if err != nil {
		if resp != nil {
			body, _ := ioutil.ReadAll(resp.Body)
			log.Printf("HTTP status: %d", resp.StatusCode)
			log.Printf("HTTP headers: %v", resp.Header)
			log.Printf("HTTP body: %s", string(body))
		}
		return nil, fmt.Errorf("dial: %v", err)
	}
	return conn, nil
}
