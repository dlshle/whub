package http

import (
	"io/ioutil"
	"net/http"
	whttp "wsdk/relay_common/http"
	"wsdk/relay_common/messages"
)

const (
	BearerTokenLen = len("Bearer ")
	AuthHeaderKey  = "Authorization"
)

// TransformRequest http request standard: header[from] = fromId, url = service url, content = body
func TransformRequest(r *http.Request) (messages.IMessage, error) {
	if r.Header["Whr"] != nil && len(r.Header["Whr"]) > 0 {
		encoded, err := whttp.EncodeToWHTTPRequestJson(r)
		if err != nil {
			return nil, err
		}
		return messages.DraftMessage(r.RemoteAddr, "", r.URL.String(), messages.MessageTypeServiceRequest, encoded), nil
	}
	var from, to, url string
	if r.Header[AuthHeaderKey] != nil && len(r.Header[AuthHeaderKey]) > 0 && len(r.Header[AuthHeaderKey][0]) > BearerTokenLen {
		// from should only be the auth token represents a client
		from = r.Header[AuthHeaderKey][0][BearerTokenLen:]
	}
	if r.Header["To"] != nil && len(r.Header["To"]) > 0 {
		to = r.Header["To"][0]
	}
	url = r.URL.Path
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	return messages.DraftMessage(from, to, url, messages.MessageTypeServiceRequest, body), nil
}
