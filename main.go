package main

import (
	"log"
	"net/http"

	"github.com/LoreKeep-2017/chatServer/auth"
	"github.com/LoreKeep-2017/chatServer/chat"
)

func main() {
	log.SetFlags(log.Lshortfile)

	// websocket serverg
	server := chat.NewServer()
	go server.Listen()

	// static files
	http.Handle("/", http.FileServer(http.Dir("webroot")))

	http.HandleFunc("/api/v1/register/", auth.RegisterHandler)
	http.HandleFunc("/api/v1/login/", server.LoginHandler)
	http.HandleFunc("/api/v1/loggedin/", auth.LoggedinHandler)
	http.HandleFunc("/api/v1/logout/", auth.LogoutHandler)
	http.HandleFunc("/api/v1/greating/", auth.GreatingHandler)
	http.HandleFunc("/api/v1/diff/", chat.DiffHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
