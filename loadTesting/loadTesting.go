package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
	// "golang.org/x/net/websocket"
)

const (
	GOROUTINNUM = 1
	TestURL     = "139.59.139.151:8080"
)

type mess struct {
	Type   string `json:"type"`
	Action string `json:"action"`
	Body   body   `json:"body"`
}
type body struct {
	Author string `json:"author"`
	Body   string `json:"body"`
}

var addr = flag.String("addr", TestURL, "http service address")
var str = "{\"type\":\"client\",\"action\" : \"sendFirstMessage\",\"body\" : {\"author\" : \"client\",\"body\": \"сообщение\",}}"

// var

func test() {
	flag.Parse()
	log.SetFlags(0)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/api/v1/client/"}
	log.Printf("connecting to %s", u.String())

	// c, err := websocket.Dial("ws://139.59.139.151:8080/api/v1/client/", "", "http://139.59.139.151")
	// ws, err := websocket.Dial(fmt.Sprintf("ws://%s/api/v1/client", addr), "", fmt.Sprintf("http://%s/", address))
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	//defer c.Close()
	// b := new(bytes.Buffer)
	// js := mess{"client", "sendFirstMessage", body{"client", "сsssss"}}
	// jsString, _ := json.Marshal(js)
	//json.NewEncoder(b).Encode(js)

	ticker := time.NewTicker(time.Millisecond * 5000)

	for t := range ticker.C {
		fmt.Println("Tick at", t)
		// fmt.Println(b)
		// c.WriteJSON(jsString)
		websocket.WriteJSON(c, []byte(str))
		// websocket.JSON.Send(c, jsString)
		if err != nil {
			log.Println("write:", err)
			return
		}
	}

}

func main() {
	donech := make(chan bool)
	for i := 0; i < GOROUTINNUM; i++ {
		go test()
	}
	<-donech

}
