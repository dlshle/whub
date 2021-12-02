package auth_service

import (
	"encoding/json"
	"fmt"
)

type LoginPayload struct {
	Id       string `json:"id"`
	Password string `json:"password"`
}

func UnmarshallLoginPayload(data []byte) (LoginPayload, error) {
	var model LoginPayload
	err := json.Unmarshal(data, &model)
	return model, err
}

func MarshallLoginResponse(token string) []byte {
	return ([]byte)(fmt.Sprintf("{\"token\":\"%s\"}", token))
}
