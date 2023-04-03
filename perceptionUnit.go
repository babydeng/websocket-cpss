package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"io"
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
var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
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
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	mqttClient := mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	<-c
}
