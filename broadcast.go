package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"os"
	"time"
)

var PORT = ":1234"
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 用于安全问题，可检查头 限制访问client
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome!\n")
	fmt.Fprintf(w, "Please use /ws for WebSocket!")
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Connection from:", r.Host)
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrader.Upgrade:", err)
		return
	}
	defer ws.Close()
	for {
		mt, message, err := ws.ReadMessage()
		if err != nil {
			log.Println("From", r.Host, "read", err)
			break
		}
		log.Print("Received: ", string(message))
		err = ws.WriteMessage(mt, message)
		if err != nil {
			log.Println("WriteMessage:", err)
			break
		}
	}
}

func main() {
	arguments := os.Args
	if len(arguments) != 1 {
		PORT = ":" + arguments[1]
	}
	mux := http.NewServeMux()
	s := &http.Server{
		Addr:         PORT,
		Handler:      mux,
		IdleTimeout:  10 * time.Second,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	}
	mux.Handle("/", http.HandlerFunc(rootHandler))
	mux.Handle("/ws", http.HandlerFunc(wsHandler))

	log.Println("Listening to TCP Port", PORT)
	err := s.ListenAndServe()
	if err != nil {
		log.Println(err)
		return
	}
}
