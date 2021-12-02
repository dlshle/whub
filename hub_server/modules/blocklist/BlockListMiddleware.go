package blocklist

import (
	"fmt"
	"strings"
	"whub/hub_common/connection"
	"whub/hub_common/messages"
	"whub/hub_common/service"
	"whub/hub_server/context"
	"whub/hub_server/middleware"
	"whub/hub_server/module_base"
)

const (
	BlockListMiddlewareId = "block_list"
	BlockListPriority     = 1
)

type BlockListMiddleware struct {
	*middleware.ServerMiddleware
	IBlockListModule `module:""`
}

func (m *BlockListMiddleware) Init() error {
	m.ServerMiddleware = middleware.NewServerMiddleware(BlockListMiddlewareId, BlockListPriority)
	err := module_base.Manager.AutoFill(m)
	if err != nil {
		panic(err)
	}
	return nil
}

func (m *BlockListMiddleware) Run(conn connection.IConnection, request service.IServiceRequest) service.IServiceRequest {
	ipPortArr := strings.Split(conn.Address(), ":")
	if len(ipPortArr) != 2 {
		request.Resolve(messages.NewErrorResponse(request, context.Ctx.Server().Id(), messages.MessageTypeSvcInternalError, "can not parse remote address"))
		return nil
	}
	ipAddr := ipPortArr[0]
	exist, err := m.Has(ipAddr)
	if err != nil {
		m.Logger().Printf("unable to check if address %s should be blocked due to %s", ipAddr, err.Error())
	}
	if exist {
		request.Resolve(messages.NewErrorResponse(request, context.Ctx.Server().Id(), messages.MessageTypeSvcForbiddenError, fmt.Sprintf("address %s has been blocklisted", ipAddr)))
		return nil
	}
	return request
}
