package client_management

import "encoding/json"

type DeleteClientsPayload struct {
	ids []string `json:"ids"`
}

func UnmarshalDeleteClientsPayload(payload []byte) (DeleteClientsPayload, error) {
	var deleteClientsPayload DeleteClientsPayload
	err := json.Unmarshal(payload, &deleteClientsPayload)
	return deleteClientsPayload, err
}
