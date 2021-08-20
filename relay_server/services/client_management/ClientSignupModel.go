package client_management

import "encoding/json"

type ClientSignupModel struct {
	Id          string `json:"id"`
	Description string `json:"description"`
	Password    string `json:"password"`
}

func UnmarshallClientSignupModel(data []byte) (ClientSignupModel, error) {
	var model ClientSignupModel
	err := json.Unmarshal(data, &model)
	return model, err
}
