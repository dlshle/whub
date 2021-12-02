package throttle

import (
	"fmt"
	"time"
	"whub/common/logger"
	"whub/hub_common/throttling"
	"whub/hub_server/config"
	"whub/hub_server/module_base"
	"whub/hub_server/modules/middleware_manager"
)

/*
 * Server side throttling strategy
 * There will be 2 throttling test be conducted on one request: general address throttling and domain specific throttling(service/client/api).
 * For general address throttling, there will be 300/min. That means, one address can only have 300 requests per minute.
 * Domain specific throttling will be tested after the general address throttling. However, if general address throttling failed, it will return error right away.
 */

const (
	ID                   = "RequestThrottle"
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

type IRequestThrottleModule interface {
	Hit(group ThrottleGroup, id string) (throttling.ThrottleRecord, error)
	GetRequestThrottleGroup(clientId string) ThrottleGroup
}

type RequestThrottleModule struct {
	*module_base.ModuleBase
	controller throttling.IThrottleController
	logger     *logger.SimpleLogger
}

func (m *RequestThrottleModule) Init() error {
	m.ModuleBase = module_base.NewModuleBase(ID, nil)
	m.controller = throttling.NewThrottleController(m.Logger())
	m.logger = m.Logger()
	return nil
}

func (m *RequestThrottleModule) OnLoad() {
	if err := middleware_manager.RegisterMiddleware(new(RequestAddressThrottleMiddleware)); err != nil {
		m.Logger().Printf("unable to register throttle middleware due to %s", err.Error())
	}
	m.ModuleBase.OnLoad()
}

func (m *RequestThrottleModule) Hit(group ThrottleGroup, id string) (record throttling.ThrottleRecord, err error) {
	// address throttling
	assembledThrottleId := m.assembleThrottleId(group.ThrottleLevel, id)
	return m.controller.Hit(assembledThrottleId, group.Limit, group.WindowDuration)
}

func (m *RequestThrottleModule) GetRequestThrottleGroup(clientId string) ThrottleGroup {
	// TODO this really depends on specific request/client
	return DefaultThrottleGroup
}

func (m *RequestThrottleModule) assembleThrottleId(throttleLevel uint8, id string) string {
	return fmt.Sprintf("%d-%s", throttleLevel, id)
}
