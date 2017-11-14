package auth

import "net/http"

func GreatingHandler(response http.ResponseWriter, request *http.Request) {
	response.WriteHeader(http.StatusOK)
	response.Write([]byte(GREATING))
}
