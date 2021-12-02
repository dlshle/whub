package auth

import "encoding/json"

type RefreshTokenMessageBody struct {
	Ttl int64 `json:"ttl"` // in milliseconds
}

func UnmarshallRefreshTokenMessageBody(data []byte) (RefreshTokenMessageBody, error) {
	var body RefreshTokenMessageBody
	err := json.Unmarshal(data, &body)
	return body, err
}
