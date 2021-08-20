package client_management

import "encoding/json"

type ClientLoginModel struct {
	Id       string `json:"id"`
	Password string `json:"password"`
}

func UnmarshallClientLoginModel(data []byte) (ClientLoginModel, error) {
	var model ClientLoginModel
	err := json.Unmarshal(data, &model)
	return model, err
}
