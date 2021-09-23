package services

import (
	"encoding/json"
	"strings"
	http_client "wsdk/common/http"
	"wsdk/relay_client"
	"wsdk/relay_client/context"
	"wsdk/relay_common/http"
	"wsdk/relay_common/roles"
	"wsdk/relay_common/service"
)

const HTTPClientServiceId = "http"
const (
	HTTPClientServiceRouteEcho = "/*path"
)

type HTTPClientService struct {
	relay_client.IClientService
	httpClient http_client.IClientPool
}

func (s *HTTPClientService) Init(server roles.ICommonServer) (err error) {
	defer func() {
		s.Logger().Println("service has been initiated with err ", err)
	}()
	s.IClientService = relay_client.NewClientService(HTTPClientServiceId, "simply echo messages", service.ServiceAccessTypeBoth, service.ServiceExecutionSync, server)
	s.httpClient = context.Ctx.HTTPClient()
	return s.InitHandlers(service.NewRequestHandlerMapBuilder().Get(HTTPClientServiceRouteEcho, s.Request).Build())
}

func (s *HTTPClientService) Request(request service.IServiceRequest, pathParams map[string]string, queryParams map[string]string) error {
	whr, err := http.DecodeToWHttpRequest(request.Payload())
	if err != nil {
		s.Logger().Println("unable to unmarshall WHTTPRequest from message", (string)(request.Payload()))
		return err
	}
	path := s.assembleRequestUrl(pathParams["path"], queryParams)
	httpRequest, err := http.FromWHTTPRequest(path, whr)
	if err != nil {
		s.Logger().Printf("unable to transfer %s to http request", whr)
		return err
	}
	resp := s.httpClient.Request(httpRequest)
	s.Logger().Printf("response to %v: %v", whr, resp)
	marshalled, err := json.Marshal(resp)
	if err != nil {
		s.Logger().Printf("unable to marshall %v due to %s", resp, err.Error())
		return err
	}
	s.ResolveByResponse(request, marshalled)
	return nil
}

func (s *HTTPClientService) assembleRequestUrl(path string, qParams map[string]string) string {
	var builder strings.Builder
	builder.WriteString("http://localhost:8888/")
	builder.WriteString(path)
	if len(qParams) == 0 {
		return builder.String()
	}
	builder.WriteByte('?')
	for k, v := range qParams {
		builder.WriteString(k)
		builder.WriteByte('=')
		builder.WriteString(v)
		builder.WriteByte('&')
	}
	return builder.String()[:builder.Len()-1]
}
