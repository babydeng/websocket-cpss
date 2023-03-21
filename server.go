package main

import (
	"encoding/json"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"time"
)

type Data struct {
	Name string `json:"name"`
	Time int64  `json:"time"`
	Num  int    `json:"num"`
}
type MyConn struct {
	// The websocket connection.
	ws *websocket.Conn

	// Buffered channel of outbound messages.
	send chan bool
}

func newConn() *MyConn {
	return &MyConn{
		ws:   nil,
		send: make(chan bool),
	}
}

type Hub struct {
	// Registered connections. That's a connection pool
	connections map[*MyConn]bool
}

func newHub() *Hub {
	return &Hub{

		connections: make(map[*MyConn]bool),
	}
}

var data = new(Data)
var persons = make(map[string]bool)
var hub = newHub()

/*
WS Params Begin
*/
var WS_PORT = ":1234"
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 用于安全问题，可检查头 限制访问client
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

/*
	WS Params End
*/

/*
WS Params Begin
*/
var MQTT_HOST = "10.177.29.226"
var MQTT_PORT = 1883

/*
	WS Params End
*/

/*
MQTT Handler Begin
*/
var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	if msg.Topic() == "face-re" {
		data.Name = string(msg.Payload())
		_, ok := persons[data.Name]
		if !ok {
			persons[data.Name] = true
			data.Time = time.Now().Unix()
			data.Num = GetNum(persons)
			for c := range hub.connections {
				select {
				case c.send <- true:
				default:
					delete(hub.connections, c)
					close(c.send)
					log.Println("Delete conn from hub!")
				}
			}
		}

	}

	fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("Connected")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Connect lost: %v", err)
}

/*
	MQTT Handler End
*/

/*
Utils Begin
*/
func GetNum(persons map[string]bool) int {
	i := 0
	for range persons {
		//fmt.Printf("The update data: %s %v\n", k,value)
		i++
	}
	return i
}

func Init() {
	old := time.Now().Unix()
	for {
		new := time.Now().Unix()
		if !InSameDay(old, new) {

			old = new
			persons = make(map[string]bool)
			fmt.Printf("clear \n")
		}
		time.Sleep(time.Minute * 30)
	}
}

func InSameDay(t1, t2 int64) bool {
	y1, m1, d1 := time.Unix(t1, 0).Date()
	y2, m2, d2 := time.Unix(t2, 0).Date()

	return y1 == y2 && m1 == m2 && d1 == d2
}

/*
	Utils End
*/

//func publish(client mqtt.Client) {
//	num := 10
//	for i := 0; i < num; i++ {
//		text := fmt.Sprintf("Message %d", i)
//		token := client.Publish("face-re", 0, false, text)
//		token.Wait()
//		time.Sleep(time.Second)
//	}
//}

func sub(client mqtt.Client) {
	topic := "face-re"
	token := client.Subscribe(topic, 1, nil)
	token.Wait()
	fmt.Printf("Subscribed to topic: %s\n", topic)
}

/*
	MQTT Handler End
*/

func rootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome!\n")
	fmt.Fprintf(w, "Please use /ws for WebSocket!")
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn := newConn()
	log.Println("Connection from:", r.Host)
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrader.Upgrade:", err)
		return
	}

	conn.ws = ws
	hub.connections[conn] = true

	for {
		select {
		case <-conn.send:
			// fmt.Printf("%T", res)
			message, err := json.Marshal(&data)
			err = ws.WriteMessage(1, message)
			if err != nil {
				log.Println("WriteMessage:", err)
				ws.Close()
				log.Println("Disconnect")
				return
			}
		}

	}

}

func main() {
	// WS Init
	mux := http.NewServeMux()
	s := &http.Server{
		Addr:         WS_PORT,
		Handler:      mux,
		IdleTimeout:  10 * time.Second,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	}

	// MQTT Init
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", MQTT_HOST, MQTT_PORT))
	opts.SetClientID("go_mqtt_client")
	// opts.SetUsername("emqx")
	// opts.SetPassword("public")

	// WS handler
	mux.Handle("/", http.HandlerFunc(rootHandler))
	mux.Handle("/ws", http.HandlerFunc(wsHandler))
	log.Println("Listening to TCP Port", WS_PORT)

	// MQTT handler
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	go Init()
	sub(client)

	err := s.ListenAndServe()
	if err != nil {
		log.Println(err)
		return
	}
}
