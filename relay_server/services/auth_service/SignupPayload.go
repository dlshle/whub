package auth_service

import "encoding/json"

type SignupPayload struct {
	Id          string `json:"id"`
	Description string `json:"description"`
	Password    string `json:"password"`
}

func UnmarshallClientSignupModel(data []byte) (SignupPayload, error) {
	var model SignupPayload
	err := json.Unmarshal(data, &model)
	return model, err
}
