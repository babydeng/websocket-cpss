package main

import (
	"bytes"
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
        "objectId": "plmm",
        "data": {
            "signed_in_count": 1
        }
    }
}`)

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

func main() {
	url := "http://10.176.34.90:9310/perceptionEvent/updateDatabase"
	fmt.Println("URL:>", url)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")

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

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Headers:", resp.Header)
	body, _ := io.ReadAll(resp.Body)
	fmt.Println("response Body:", string(body))

	<-c
}
