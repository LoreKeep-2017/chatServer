package auth

const (
	GREATING = "Здаровствуйте! Как я могу Вам помочь?"
)

type OperatorId struct {
	Id       int    `json:"id,omitempty"`
	Login    string `json:"login,omitempty"`
	Password string `json:"password,omitempty"`
	FIO      string `json:"fio,omitempty"`
}

type Greating struct {
	Greating string `json:"greating"`
}
