package main

import (
	"log"
	"net/http"

	"github.com/LoreKeep-2017/chatServer/auth"
	"github.com/LoreKeep-2017/chatServer/chat"

	"github.com/gorilla/handlers"
)

func main() {
	log.SetFlags(log.Lshortfile)

	// websocket serverg
	server := chat.NewServer()
	go server.Listen()

	// static files
	http.Handle("/", http.FileServer(http.Dir("webroot")))

	//r := mux.NewRouter()

	server.Router.HandleFunc("/api/v1/register/", auth.RegisterHandler)
	server.Router.HandleFunc("/api/v1/login/", server.LoginHandler)
	server.Router.HandleFunc("/api/v1/loggedin/", server.LoggedinHandler)
	server.Router.HandleFunc("/api/v1/logout/", auth.LogoutHandler)
	server.Router.HandleFunc("/api/v1/greating/", auth.GreatingHandler)
	server.Router.HandleFunc("/api/v1/diff/", chat.DiffHandler)
	// AllowedOrigins(origins []string) CORSOption
	corsOPT := handlers.AllowedOrigins([]string{"*"})
	log.Fatal(http.ListenAndServe(":8080", handlers.CORS(corsOPT)(server.Router)))
}
