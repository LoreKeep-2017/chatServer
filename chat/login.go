package chat

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/LoreKeep-2017/chatServer/auth"
	"github.com/LoreKeep-2017/chatServer/db"
)

func (s *Server) LoginHandler(response http.ResponseWriter, request *http.Request) {
	decoder := json.NewDecoder(request.Body)
	var operator auth.OperatorId
	err := decoder.Decode(&operator)
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		response.Write([]byte("500 - cannot parse json!"))
		return
	}
	defer request.Body.Close()
	if operator.Login != "" && operator.Password != "" {
		// .. check credentials ..
		dbinfo := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", db.DB_USER, db.DB_PASSWORD, db.DB_NAME)
		db, _ := sql.Open("postgres", dbinfo)
		id := 0
		fio := ""
		err := db.QueryRow("SELECT id, fio FROM operator where nickname=$1 and password=$2",
			operator.Login, operator.Password).Scan(&id, &fio)
		if err != nil {
			response.WriteHeader(http.StatusNotFound)
			response.Write([]byte("404 - wrong login or password!"))
		} else {
			if _, ok := s.operators[id]; ok {
				response.WriteHeader(http.StatusConflict)
				response.Write([]byte("409 - session alredy exist!"))
				return
			}
			if id > 0 {
				operator.Id = id
				operator.Password = ""
				operator.FIO = fio
				auth.SetSession(operator, response, id)
			} else {
				response.WriteHeader(http.StatusNotFound)
				response.Write([]byte("404 - wrong login or password!"))
			}
		}

	} else {
		response.WriteHeader(http.StatusBadRequest)
		response.Write([]byte("400 - empty login or password!"))
	}
}

func (s *Server) checkSession(response http.ResponseWriter, cookie *http.Cookie) {
	var value OperatorId
	if err := auth.CookieHandler.Decode(cookie.Name, cookie.Value, &value); err == nil {
		if _, ok := s.operators[value.Id]; ok {
			response.WriteHeader(http.StatusConflict)
			response.Write([]byte("409 - session alredy exist!"))
			return
		}
		js, _ := json.Marshal(value)
		response.WriteHeader(http.StatusOK)
		response.Write(js)
	} else {
		response.WriteHeader(http.StatusForbidden)
		response.Write([]byte("403 - Forbidden! "))
	}
}

func (s *Server) LoggedinHandler(response http.ResponseWriter, request *http.Request) {
	cookie, err := request.Cookie("session")
	if err == nil {
		s.checkSession(response, cookie)
	} else {
		response.WriteHeader(http.StatusUnauthorized)
		response.Write([]byte("401 - Unauthorized!"))
	}

}
