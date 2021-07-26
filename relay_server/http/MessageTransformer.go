package http

import (
	"io/ioutil"
	"net/http"
	"wsdk/relay_common/messages"
)

// TransformRequest http request standard: header[from] = fromId, url = service url, content = body
func TransformRequest(r *http.Request) (*messages.Message, error) {
	var from, to, url string
	if r.Header["from"] != nil && len(r.Header["from"]) > 0 {
		from = r.Header["from"][0]
	}
	if r.Header["to"] != nil && len(r.Header["to"]) > 0 {
		to = r.Header["to"][0]
	}
	url = r.URL.Path
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	return messages.DraftMessage(from, to, url, messages.MessageTypeServiceRequest, body), nil
}
