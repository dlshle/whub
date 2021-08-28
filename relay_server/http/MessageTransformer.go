package http

import (
	"errors"
	"io/ioutil"
	"net/http"
	whttp "wsdk/relay_common/http"
	"wsdk/relay_common/messages"
	"wsdk/relay_server/core/auth"
)

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
	if r.Header["To"] != nil && len(r.Header["To"]) > 0 {
		to = r.Header["To"][0]
	}
	url = r.URL.Path
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	return messages.DraftMessage(from, to, url, msgType, body), nil
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
