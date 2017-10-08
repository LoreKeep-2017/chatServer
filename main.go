package main

import (
	"log"
	"net/http"
	"github.com/LoreKeep-2017/chatServer"
)


func main() {
	log.SetFlags(log.Lshortfile)

	// websocket server
	server := chat.NewServer()
	go server.Listen()

	// static files
	http.Handle("/", http.FileServer(http.Dir("webroot")))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
