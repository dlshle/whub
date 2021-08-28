package auth

const (
	BearerTokenLen = len("Bearer ")
	AuthHeaderKey  = "Authorization"
)

func GetTrimmedHTTPToken(header map[string][]string) string {
	if header[AuthHeaderKey] != nil && len(header[AuthHeaderKey]) > 0 && len(header[AuthHeaderKey][0]) > BearerTokenLen {
		return header[AuthHeaderKey][0][BearerTokenLen:]
	}
	return ""
}
