package auth

import (
	"encoding/json"
	"net/http"
)

func GreatingHandler(response http.ResponseWriter, request *http.Request) {
	greating := Greating{Greating: GREATING}
	js, _ := json.Marshal(greating)
	response.Header().Set("Access-Control-Allow-Origin", "*")
	response.WriteHeader(http.StatusOK)
	response.Write(js)
}
