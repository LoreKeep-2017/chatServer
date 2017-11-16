package auth

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/LoreKeep-2017/chatServer/db"
	"github.com/gorilla/securecookie"
)

var CookieHandler = securecookie.New(
	securecookie.GenerateRandomKey(64),
	securecookie.GenerateRandomKey(32))

func SetSession(operator OperatorId, response http.ResponseWriter, id int) {
	// value := map[string]string{
	// 	"name": userName,
	// }
	if encoded, err := CookieHandler.Encode("session", operator); err == nil {
		cookie := &http.Cookie{
			Name:  "session",
			Value: encoded,
			Path:  "/",
		}
		http.SetCookie(response, cookie)
		response.WriteHeader(http.StatusOK)
		response.Header().Add("Content-Type", "application/json")
		//operator := OperatorId{id, userName, ""}
		js, _ := json.Marshal(operator)
		response.Write(js)
	}
}

func clearSession(response http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:   "session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	http.SetCookie(response, cookie)
	response.WriteHeader(http.StatusOK)
	response.Write([]byte("200 - Ok!"))
}

// func checkSession(response http.ResponseWriter, cookie *http.Cookie) {
// 	var value OperatorId
// 	if err := cookieHandler.Decode(cookie.Name, cookie.Value, &value); err == nil {
// 		js, _ := json.Marshal(value)
// 		response.WriteHeader(http.StatusOK)
// 		response.Write(js)
// 	} else {
// 		response.WriteHeader(http.StatusForbidden)
// 		response.Write([]byte("403 - Forbidden! "))
// 	}
// }

// func LoginHandler(response http.ResponseWriter, request *http.Request) {
// 	decoder := json.NewDecoder(request.Body)
// 	var operator OperatorId
// 	err := decoder.Decode(&operator)
// 	if err != nil {
// 		response.WriteHeader(http.StatusInternalServerError)
// 		response.Write([]byte("500 - cannot parse json!"))
// 		return
// 	}
// 	defer request.Body.Close()
// 	if operator.Login != "" && operator.Password != "" {
// 		// .. check credentials ..
// 		dbinfo := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", db.DB_USER, db.DB_PASSWORD, db.DB_NAME)
// 		db, _ := sql.Open("postgres", dbinfo)
// 		id := 0
// 		fio := ""
// 		err := db.QueryRow("SELECT id, fio FROM operator where nickname=$1 and password=$2",
// 			operator.Login, operator.Password).Scan(&id, &fio)
// 		if err != nil {
// 			response.WriteHeader(http.StatusNotFound)
// 			response.Write([]byte("404 - wrong login or password!"))
// 		} else {
// 			if id > 0 {
// 				operator.Id = id
// 				operator.Password = ""
// 				operator.FIO = fio
// 				setSession(operator, response, id)
// 			} else {
// 				response.WriteHeader(http.StatusNotFound)
// 				response.Write([]byte("404 - wrong login or password!"))
// 			}
// 		}
//
// 	} else {
// 		response.WriteHeader(http.StatusBadRequest)
// 		response.Write([]byte("400 - empty login or password!"))
// 	}
// }

// func LoggedinHandler(response http.ResponseWriter, request *http.Request) {
// 	cookie, err := request.Cookie("session")
// 	if err == nil {
// 		checkSession(response, cookie)
// 	} else {
// 		response.WriteHeader(http.StatusUnauthorized)
// 		response.Write([]byte("401 - Unauthorized!"))
// 	}
//
// }

func LogoutHandler(response http.ResponseWriter, request *http.Request) {
	clearSession(response)
}

func RegisterHandler(response http.ResponseWriter, request *http.Request) {
	decoder := json.NewDecoder(request.Body)
	var operator OperatorId
	err := decoder.Decode(&operator)
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		response.Write([]byte("500 - cannot parse json!"))
		return
	}
	defer request.Body.Close()
	if operator.Login != "" && operator.Password != "" {
		dbinfo := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", db.DB_USER, db.DB_PASSWORD, db.DB_NAME)
		db, _ := sql.Open("postgres", dbinfo)
		_, err := db.Query(`insert into operator(nickname, password) values($1, $2)`, operator.Login, operator.Password)
		if err != nil {
			response.WriteHeader(http.StatusInternalServerError)
			response.Write([]byte("500 - StatusInternalServerError! " + err.Error()))
		} else {
			response.WriteHeader(http.StatusOK)
			response.Write([]byte("200 - Ok!"))
		}

	} else {
		response.WriteHeader(http.StatusBadRequest)
		response.Write([]byte("400 - wrong register data!"))
	}
}
