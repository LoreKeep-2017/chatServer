package auth

const (
	GREATING = "Здарова, чертила!"
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
