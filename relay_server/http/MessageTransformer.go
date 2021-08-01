package http

import (
	"io/ioutil"
	"net/http"
	whttp "wsdk/relay_common/http"
	"wsdk/relay_common/messages"
)

// TransformRequest http request standard: header[from] = fromId, url = service url, content = body
func TransformRequest(r *http.Request) (*messages.Message, error) {
	if r.Header["Whr"] != nil && len(r.Header["Whr"]) > 0 {
		encoded, err := whttp.EncodeToWHTTPRequestJson(r)
		if err != nil {
			return nil, err
		}
		return messages.DraftMessage(r.RemoteAddr, "", r.URL.String(), messages.MessageTypeServiceRequest, encoded), nil
	}
	var from, to, url string
	if r.Header["From"] != nil && len(r.Header["From"]) > 0 {
		from = r.Header["From"][0]
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
