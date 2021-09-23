package throttle

import (
	"fmt"
	"time"
	"wsdk/common/logger"
	"wsdk/relay_common/throttling"
	"wsdk/relay_server/config"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
)

/*
 * Server side throttling strategy
 * There will be 2 throttling test be conducted on one request: general address throttling and domain specific throttling(service/client/api).
 * For general address throttling, there will be 300/min. That means, one address can only have 300 requests per minute.
 * Domain specific throttling will be tested after the general address throttling. However, if general address throttling failed, it will return error right away.
 */

const (
	ThrottleLevelService = 0 // Per client per service or per address per service?
	ThrottleLevelClient  = 1
	ThrottleLevelApi     = 2 // Per client per service API or per address per service API?
	ThrottleLevelAddress = 3

	BlockListExceedingHitFactor = 3
)

var DefaultThrottleGroup ThrottleGroup
var AddressThrottleGroup ThrottleGroup

func initGlobalVariables() {
	throttleConfigs := config.Config.ThrottleConfigs
	addressWindow := throttleConfigs["address"].Window
	addressLimit := throttleConfigs["address"].Limit
	DefaultThrottleGroup = ThrottleGroup{
		ThrottleLevel:  ThrottleLevelClient,
		WindowDuration: time.Minute,
		Limit:          150,
	}
	AddressThrottleGroup = ThrottleGroup{
		ThrottleLevel:  ThrottleLevelAddress,
		WindowDuration: time.Minute,
		Limit:          300,
	}
	if addressWindow > 0 && addressWindow < 600 {
		AddressThrottleGroup.WindowDuration = time.Duration(addressWindow) * time.Second
	}
	if addressLimit > 0 && addressLimit < 500 {
		AddressThrottleGroup.Limit = addressLimit
	}
}

func init() {
	initGlobalVariables()
}

type ThrottleGroup struct {
	ThrottleLevel  uint8
	WindowDuration time.Duration
	Limit          int
}

type IRequestThrottleController interface {
	Hit(group ThrottleGroup, id string) (throttling.ThrottleRecord, error)
	GetRequestThrottleGroup(clientId string) ThrottleGroup
}

type RequestThrottleController struct {
	controller throttling.IThrottleController
	logger     *logger.SimpleLogger
}

func NewRequestThrottleController() IRequestThrottleController {
	logger := context.Ctx.Logger().WithPrefix("[RequestThrottleController]")
	return &RequestThrottleController{
		controller: throttling.NewThrottleController(logger),
		logger:     logger,
	}
}

func (c *RequestThrottleController) Hit(group ThrottleGroup, id string) (record throttling.ThrottleRecord, err error) {
	// address throttling
	assembledThrottleId := c.assembleThrottleId(group.ThrottleLevel, id)
	return c.controller.Hit(assembledThrottleId, group.Limit, group.WindowDuration)
}

func (c *RequestThrottleController) GetRequestThrottleGroup(clientId string) ThrottleGroup {
	// TODO this really depends on specific request/client
	return DefaultThrottleGroup
}

func (c *RequestThrottleController) assembleThrottleId(throttleLevel uint8, id string) string {
	return fmt.Sprintf("%d-%s", throttleLevel, id)
}

func Load() error {
	return container.Container.Singleton(func() IRequestThrottleController {
		return NewRequestThrottleController()
	})
}
