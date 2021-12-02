package throttle

import (
	"fmt"
	"strings"
	"whub/hub_common/connection"
	"whub/hub_common/messages"
	"whub/hub_common/service"
	"whub/hub_server/context"
	"whub/hub_server/middleware"
	"whub/hub_server/module_base"
	"whub/hub_server/modules/blocklist"
)

const (
	RequestAddressThrottleMiddlewareId       = "request_throttle"
	RequestAddressThrottleMiddlewarePriority = 2
	AddressThrottleWindowExpiresContextKey   = "throttle-addr-window-expire"
	AddressThrottleHitRemainsContextKey      = "throttle-addr-hit-remains"
)

type RequestAddressThrottleMiddleware struct {
	*middleware.ServerMiddleware
	IRequestThrottleModule     `module:""`
	blocklist.IBlockListModule `module:""`
}

func (m *RequestAddressThrottleMiddleware) Init() error {
	m.ServerMiddleware = middleware.NewServerMiddleware(RequestAddressThrottleMiddlewareId, RequestAddressThrottleMiddlewarePriority)
	return module_base.Manager.AutoFill(m)
}

func (m *RequestAddressThrottleMiddleware) Run(conn connection.IConnection, request service.IServiceRequest) service.IServiceRequest {
	splitAddr := strings.Split(conn.Address(), ":")
	if len(splitAddr) != 2 {
		request.Resolve(messages.NewErrorResponse(request, context.Ctx.Server().Id(), messages.MessageTypeSvcInternalError, "can not parse remote address"))
		return nil
	}
	ipAddr := splitAddr[0]
	record, err := m.Hit(AddressThrottleGroup, ipAddr)
	if err != nil {
		m.Logger().Printf("unable to get throttle hit count for address %s due to %s", ipAddr, err.Error())
		// if can not check throttle status, skip to the next middleware
		return request
	}
	remains := record.Limit - record.HitsUnderWindow
	if remains < 0 {
		shouldDemote, _ := m.checkAndDemoteAddrToBlockList(remains, record.Limit, ipAddr)
		if shouldDemote {
			request.Resolve(messages.NewErrorResponse(request, context.Ctx.Server().Id(), messages.MessageTypeSvcForbiddenError, fmt.Sprintf("address %s has been blocklisted", ipAddr)))
			return nil
		}
		remains = 0
	}
	request.SetContext(AddressThrottleWindowExpiresContextKey, record.WindowExpiration.Format("2006-01-02 15:04:05"))
	request.SetContext(AddressThrottleHitRemainsContextKey, remains)
	return request
}

func (m *RequestAddressThrottleMiddleware) checkAndDemoteAddrToBlockList(remains, limit int, addr string) (shouldDemote bool, err error) {
	shouldDemote = false
	if remains < (-1 * limit * BlockListExceedingHitFactor) {
		shouldDemote = true
		err = m.DemoteByAddr(addr)
		if err != nil {
			m.Logger().Printf("unable to demote addr %s due to %s", addr, err.Error())
			return
		}
	}
	return
}
