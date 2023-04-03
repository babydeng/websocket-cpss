package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	_ "github.com/lib/pq"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

/*
Other Params Begin
*/
var data = []byte(`{
    "eventIdentify": {
        "eventId": "mixiaochao_boy",
        "name": "Sign_In",
        "topic": "Sign_In.selab01",
        "location": "selab01"
    },
    "timestamp": 1679849540,
    "perceptionEventType": 1,
    "eventData": {
        "location": "selab01",
        "objectId": "",
        "data": {
            "signed_in_count": 0
        }
    }
}`)

// 定义打卡数据结构体
type Attendance struct {
	ID   int64     `json:"id"`
	Name string    `json:"name"`
	Time time.Time `json:"time"`
}

// var data = Data{Num: 0, Time: time.Now().Unix(), Name: ""}
var sentNames = make(map[string]string)

/*
Other Params End
*/

/*
MQTT Params Begin
*/
var MQTT_HOST = "10.177.29.226"
var MQTT_PORT = 1883

/*
	MQTT Params End
*/

/*
MQTT Handler Begin
*/

func messagePubHandler(db *sql.DB) func(mqtt.Client, mqtt.Message) {
	return func(client mqtt.Client, msg mqtt.Message) {
		fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())

		var obj map[string]interface{}
		if err := json.Unmarshal(data, &obj); err != nil {
			panic(err)
		}
		if msg.Topic() == "face-re" {
			name := string(msg.Payload())
			today := time.Now().Format("2006-01-02")
			if sentDate, ok := sentNames[name]; ok {
				if sentDate == today {
					return
				}
			} else {
				sentNames[name] = today
				obj["eventData"].(map[string]interface{})["objectId"] = name
				obj["eventData"].(map[string]interface{})["data"].(map[string]interface{})["signed_in_count"] = 1
				obj["timestamp"] = time.Now().Unix()
				err := sendEvent(obj)
				if err != nil {
					fmt.Println(err)
					return
				}
				attendance := Attendance{Name: name, Time: time.Now()}
				_, err = db.Exec("INSERT INTO attendance (name, time) VALUES ($1, $2)", attendance.Name, attendance.Time)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Printf("插入打卡数据 ：%s，时间：%s\n", attendance.Name, attendance.Time.Format("2006-01-02 15:04:05"))
			}

		}
	}
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	sub(client)
	fmt.Println("Connected")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Connect lost: %v", err)
}

func sub(client mqtt.Client) {
	topic := "face-re"
	token := client.Subscribe(topic, 1, nil)
	token.Wait()
	fmt.Printf("Subscribed to topic: %s\n", topic)
}

/*
	MQTT Handler End
*/

func sendEvent(data map[string]interface{}) error {
	newData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	url := "http://10.176.34.90:9310/perceptionEvent/updateDatabase"
	fmt.Println("URL:>", url)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(newData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Headers:", resp.Header)
	body, _ := io.ReadAll(resp.Body)
	fmt.Println("response Body:", string(body))
	return nil
}

func main() {

	// 数据库初始化
	// 创建数据库连接
	db, err := sql.Open("postgres", "postgres://pi:123456@10.177.21.124/restapi?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 创建打卡表
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS attendance (id SERIAL PRIMARY KEY, name VARCHAR(50), time TIMESTAMP)")
	if err != nil {
		log.Fatal(err)
	}

	// 在每天的0点清空map变量
	go func() {
		for {
			now := time.Now()
			next := now.Add(time.Hour * 24)
			next = time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location())
			t := time.NewTimer(next.Sub(now))
			<-t.C
			sentNames = make(map[string]string)
			var obj map[string]interface{}
			if err := json.Unmarshal(data, &obj); err != nil {
				panic(err)
			}
			obj["eventData"].(map[string]interface{})["objectId"] = ""
			obj["eventData"].(map[string]interface{})["data"].(map[string]interface{})["signed_in_count"] = 0
			obj["timestamp"] = time.Now().Unix()
			err := sendEvent(obj)
			if err != nil {
				fmt.Println(err)
				continue
			}
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// MQTT Init
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", MQTT_HOST, MQTT_PORT))
	opts.SetClientID(time.Now().String())

	// MQTT handler
	opts.SetDefaultPublishHandler(messagePubHandler(db))
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	mqttClient := mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	<-c
}
