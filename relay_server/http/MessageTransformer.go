package http

import (
	"errors"
	"io/ioutil"
	"net/http"
	whttp "wsdk/relay_common/http"
	"wsdk/relay_common/messages"
	"wsdk/relay_server/modules/auth"
)

var reservedHeaders map[string]bool

func init() {
	reservedHeaders = make(map[string]bool)
	reservedHeaders["Whr"] = true
	reservedHeaders["From"] = true
	reservedHeaders["To"] = true
	reservedHeaders["Id"] = true
	reservedHeaders["Content-Type"] = true
	reservedHeaders["Content-Length"] = true
	reservedHeaders["Host"] = true
	reservedHeaders["User-Agent"] = true
	reservedHeaders["Accept"] = true
	reservedHeaders["Accept-Encoding"] = true
	reservedHeaders["Connection"] = true
	reservedHeaders["R-Token"] = true
}

func isWhrRequest(r *http.Request) bool {
	return r.Header["Whr"] != nil && len(r.Header["Whr"]) > 0
}

// TransformRequest http request standard: header[from] = fromId, url = service url, content = body
func TransformRequest(r *http.Request) (messages.IMessage, error) {
	msgType, err := mapHttpRequestMethodToMessageType(r.Method)
	if err != nil {
		return nil, err
	}
	if isWhrRequest(r) {
		encoded, err := whttp.EncodeToWHTTPRequestJson(r)
		if err != nil {
			return nil, err
		}
		return messages.DraftMessage(r.RemoteAddr, "", r.URL.String(), messages.MessageTypeServiceRequest, encoded), nil
	}
	var from, to, url string
	// from should only be the auth token represents a client
	from = auth.GetTrimmedHTTPToken(r.Header)
	if len(r.Header[messages.MessageHTTPHeaderTo]) > 0 {
		to = r.Header[messages.MessageHTTPHeaderTo][0]
	}
	url = r.URL.Path
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	message := messages.DraftMessage(from, to, url, msgType, body)
	message = transformHeaderFields(message, r.Header)
	return message, nil
}

func transformHeaderFields(message messages.IMessage, httpHeaders map[string][]string) messages.IMessage {
	for k, v := range httpHeaders {
		if !reservedHeaders[k] && len(v) > 0 {
			message.SetHeader(k, v[0])
		}
	}
	return message
}

func mapHttpRequestMethodToMessageType(method string) (int, error) {
	switch method {
	case http.MethodGet:
		return messages.MessageTypeServiceGetRequest, nil
	case http.MethodPut:
		return messages.MessageTypeServicePutRequest, nil
	case http.MethodPost:
		return messages.MessageTypeServicePostRequest, nil
	case http.MethodPatch:
		return messages.MessageTypeServicePatchRequest, nil
	case http.MethodDelete:
		return messages.MessageTypeServiceDeleteRequest, nil
	case http.MethodHead:
		return messages.MessageTypeServiceHeadRequest, nil
	case http.MethodOptions:
		return messages.MessageTypeServiceOptionsRequest, nil
	default:
		return -1, errors.New("unsupported http method")
	}
}
