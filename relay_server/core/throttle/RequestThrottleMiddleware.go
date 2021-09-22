package throttle

import (
	"strings"
	"wsdk/relay_common/connection"
	"wsdk/relay_common/messages"
	"wsdk/relay_common/service"
	"wsdk/relay_server/container"
	"wsdk/relay_server/core/middleware_manager"
	"wsdk/relay_server/middleware"
)

const (
	RequestAddressThrottleMiddlewareId       = "request_throttle"
	RequestAddressThrottleMiddlewarePriority = 0
	AddressThrottleWindowExpiresContextKey   = "throttle-addr-window-expire"
	AddressThrottleHitRemainsContextKey      = "throttle-addr-hit-remains"
)

type RequestAddressThrottleMiddleware struct {
	*middleware.ServerMiddleware
	IRequestThrottleController `$inject:""`
}

func (m *RequestAddressThrottleMiddleware) Init() error {
	m.ServerMiddleware = middleware.NewServerMiddleware(RequestAddressThrottleMiddlewareId, RequestAddressThrottleMiddlewarePriority)
	return container.Container.Fill(m)
}

func (m *RequestAddressThrottleMiddleware) Run(conn connection.IConnection, request service.IServiceRequest) service.IServiceRequest {
	splitAddr := strings.Split(conn.Address(), ":")
	ipAddr := splitAddr[0]
	record, err := m.Hit(AddressThrottleGroup, ipAddr)
	remains := record.Limit - record.HitsUnderWindow
	if remains < 0 {
		remains = 0
	}
	// TODO if remains < BlockListThreshold, add this addr to block list for couple of hours/days
	request.SetContext(AddressThrottleWindowExpiresContextKey, record.WindowExpiration.Format("2006-01-02 15:04:05"))
	request.SetContext(AddressThrottleHitRemainsContextKey, remains)
	if err != nil {
		request.Resolve(messages.NewErrorResponse(request, "", messages.MessageTypeSvcForbiddenError, err.Error()))
		return nil
	}
	return request
}

func Register() {
	middleware_manager.RegisterMiddleware(new(RequestAddressThrottleMiddleware))
}
