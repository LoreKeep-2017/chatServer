package auth

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/LoreKeep-2017/chatServer/db"
	"github.com/gorilla/securecookie"
)

var cookieHandler = securecookie.New(
	securecookie.GenerateRandomKey(64),
	securecookie.GenerateRandomKey(32))

func setSession(userName string, response http.ResponseWriter, id int) {
	value := map[string]string{
		"name": userName,
	}
	if encoded, err := cookieHandler.Encode("session", value); err == nil {
		cookie := &http.Cookie{
			Name:  "session",
			Value: encoded,
			Path:  "/",
		}
		http.SetCookie(response, cookie)
		response.WriteHeader(http.StatusOK)
		response.Header().Add("Content-Type", "application/json")
		operator := OperatorId{id}
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

func checkSession(response http.ResponseWriter, cookie *http.Cookie) {
	value := make(map[string]string)
	if err := cookieHandler.Decode(cookie.Name, cookie.Value, &value); err == nil {
		//fmt.Fprintf(w, "The value of foo is %q", value["foo"])
		response.WriteHeader(http.StatusOK)
		response.Write([]byte("200 - OK!"))
	} else {
		response.WriteHeader(http.StatusForbidden)
		response.Write([]byte("403 - Forbidden! "))
	}
}

func LoginHandler(response http.ResponseWriter, request *http.Request) {
	name := request.FormValue("login")
	pass := request.FormValue("password")
	if name != "" && pass != "" {
		// .. check credentials ..
		dbinfo := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", db.DB_USER, db.DB_PASSWORD, db.DB_NAME)
		db, _ := sql.Open("postgres", dbinfo)
		id := 0
		err := db.QueryRow("SELECT id FROM operator where nickname=$1 and password=$2",
			name, pass).Scan(&id)
		if err != nil {
			response.WriteHeader(http.StatusNotFound)
			response.Write([]byte("404 - wrong login or password!"))
		} else {
			if id > 0 {
				setSession(name, response, id)
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

func LoggedinHandler(response http.ResponseWriter, request *http.Request) {
	cookie, err := request.Cookie("session")
	if err == nil {
		checkSession(response, cookie)
	} else {
		response.WriteHeader(http.StatusUnauthorized)
		response.Write([]byte("401 - Unauthorized!"))
	}

}

func LogoutHandler(response http.ResponseWriter, request *http.Request) {
	clearSession(response)
}

func RegisterHandler(response http.ResponseWriter, request *http.Request) {
	name := request.FormValue("login")
	pass := request.FormValue("password")
	repass := request.FormValue("repeatPassword")
	if (name != "" && pass != "") || (repass != pass) {
		dbinfo := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", db.DB_USER, db.DB_PASSWORD, db.DB_NAME)
		db, _ := sql.Open("postgres", dbinfo)
		_, err := db.Query(`insert into operator(nickname, password) values($1, $2)`, name, pass)
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
