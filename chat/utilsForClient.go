package chat

import "encoding/json"

//// Client messages

type ClientRequest struct {
	Action  string          `json:"action"`
	RawData json.RawMessage `json:"data,omitempty"`
}

type ClientSendMessageRequest struct {
	Msg string `json:"msg"`
}

type ClientGreetingResponse struct {
	Action string `json:"action"`
	Msg    string `json:"data"`
}
