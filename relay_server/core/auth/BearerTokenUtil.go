package auth

const (
	AuthHeaderKey = "R-Token"
)

func GetTrimmedHTTPToken(header map[string][]string) string {
	if header[AuthHeaderKey] != nil && len(header[AuthHeaderKey]) > 0 {
		return header[AuthHeaderKey][0]
	}
	return ""
}
