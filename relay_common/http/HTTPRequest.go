package http

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
)

type WHttpRequest struct {
	Method     string
	Header     http.Header
	Body       []byte
	Form       url.Values
	PostForm   url.Values
	RemoteAddr string
}

func ToWHTTPRequest(r *http.Request) (*WHttpRequest, error) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	return &WHttpRequest{
		Method:     r.Method,
		Header:     r.Header,
		Body:       body,
		Form:       r.Form,
		PostForm:   r.PostForm,
		RemoteAddr: r.RemoteAddr,
	}, nil
}

func EncodeToWHTTPRequestJson(r *http.Request) ([]byte, error) {
	whr, err := ToWHTTPRequest(r)
	if err != nil {
		return nil, err
	}
	return json.Marshal(whr)
}

func FromWHTTPRequest(url string, r *WHttpRequest) (*http.Request, error) {
	httpReq, err := http.NewRequest(r.Method, url, bytes.NewBuffer(r.Body))
	if err != nil {
		return nil, err
	}
	httpReq.Header = r.Header
	httpReq.Form = r.Form
	httpReq.PostForm = r.PostForm
	httpReq.RemoteAddr = r.RemoteAddr
	return httpReq, nil
}

func DecodeToWHttpRequest(data []byte) (*WHttpRequest, error) {
	var whr WHttpRequest
	err := json.Unmarshal(data, &whr)
	return &whr, err
}
