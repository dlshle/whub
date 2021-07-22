package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"wsdk/common/logger"
	"wsdk/common/observable"
	"wsdk/common/reflect"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
)

const (
	defaultMaxListenerCount        = 1024
	defaultAsyncPoolSize           = 2048
	defaultServicePoolSize         = 1024
	defaultAsyncPoolWorkerFactor   = 32
	defaultServicePoolWorkerFactor = 16
	defaultMaxConcurrentConnection = 2048
	defaultMaxServicePerClient     = 8
	defaultGreetingMessage         = ""

	// config keys
	MaxAsyncPoolSize             = "MaxAsyncPoolSize"
	MaxServiceAsyncPoolSize      = "MaxServiceAsyncPoolSize "
	AsyncPoolWorkerFactor        = "AsyncPoolWorkerFactor "
	ServiceAsyncPoolWorkerFactor = "ServiceAsyncPoolWorkerFactor "
	MaxListenerCount             = "MaxListenerCount "
	MaxConnectionCount           = "MaxConnectionCount "
	MaxServicePerClient          = "MaxServicePerClient "
	GreetingMessage              = "GreetingMessage"
)

var configKeysMap map[string]bool

func initConfigKeysMap() {
	configKeysMap = make(map[string]bool)
	configKeysMap[MaxAsyncPoolSize] = true
	configKeysMap[MaxServiceAsyncPoolSize] = true
	configKeysMap[AsyncPoolWorkerFactor] = true
	configKeysMap[ServiceAsyncPoolWorkerFactor] = true
	configKeysMap[MaxListenerCount] = true
	configKeysMap[MaxConnectionCount] = true
	configKeysMap[MaxServicePerClient] = true
	configKeysMap[GreetingMessage] = true
}

type ServerConfig struct {
	MaxAsyncPoolSize             int `json:"maxAsyncPoolSize""`
	MaxServiceAsyncPoolSize      int `json:"maxServiceAsyncPoolSize"`
	AsyncPoolWorkerFactor        int `json:"asyncPoolWorkerFactor"`
	ServiceAsyncPoolWorkerFactor int `json:"serviceAsyncPoolWorkerFactor"`
	MaxListenerCount             int `json:"maxListenerCount"`

	MaxConnectionCount  int `json:"maxConnectionCount"`
	MaxServicePerClient int `json:"maxServicePerClient"`

	GreetingMessage string `json:"greetingMessage"`
}

func checkConfigKey(key string) error {
	if !configKeysMap[key] {
		return errors.New(fmt.Sprintf("config key (%s) is invalid", key))
	}
	return nil
}

type ServerConfigManager struct {
	configMap            map[string]interface{}
	observableUpdatedKey observable.IObservable
	logger               *logger.SimpleLogger
}

type IServerConfigManager interface {
	UpdateConfigsByJson(configJson string) error
	UpdateConfigs(config ServerConfig) error
	UpdateConfig(key string, value interface{}) error
	GetConfig(key string) interface{}
	OnConfigChange(key string, cb func(value interface{}))
}

func NewServerConfigManager() IServerConfigManager {
	return &ServerConfigManager{
		configMap:            make(map[string]interface{}),
		observableUpdatedKey: observable.NewSafeObservable(),
		logger:               context.Ctx.Logger().WithPrefix("[ServerConfigManager]"),
	}
}

func (m *ServerConfigManager) notifyConfigChange(key string) {
	m.observableUpdatedKey.Set(key)
}

func (m *ServerConfigManager) UpdateConfigsByJson(configJson string) error {
	var newConfig ServerConfig
	err := json.Unmarshal(([]byte)(configJson), &newConfig)
	if err != nil {
		return err
	}
	return m.UpdateConfigs(newConfig)
}

func (m *ServerConfigManager) UpdateConfigs(config ServerConfig) error {
	fvMap, err := reflect.GetFieldsAndValues(config)
	m.logger.Println("update configs with map", fvMap)
	if err != nil {
		return err
	}
	for k, v := range fvMap {
		err = m.UpdateConfig(k, v)
		if err != nil {
			return err
		}
	}
	m.logger.Println("updated configs: ", m.configMap)
	return nil
}

func (m *ServerConfigManager) UpdateConfig(key string, value interface{}) error {
	if err := checkConfigKey(key); err != nil {
		m.logger.Println(err)
		// return err
	}
	m.configMap[key] = value
	m.logger.Printf("config %s is set to %v", key, value)
	m.notifyConfigChange(key)
	return nil
}

func (m *ServerConfigManager) GetConfig(key string) interface{} {
	return m.configMap[key]
}

func (m *ServerConfigManager) OnConfigChange(key string, cb func(value interface{})) {
	m.observableUpdatedKey.On(func(configKey interface{}) {
		ckey := configKey.(string)
		if ckey != key {
			return
		}
		cb(m.GetConfig(ckey))
	})
}

func init() {
	initConfigKeysMap()
	defaultConfig := ServerConfig{
		MaxAsyncPoolSize:             defaultAsyncPoolSize,
		MaxServiceAsyncPoolSize:      defaultServicePoolSize,
		AsyncPoolWorkerFactor:        defaultAsyncPoolWorkerFactor,
		ServiceAsyncPoolWorkerFactor: defaultServicePoolWorkerFactor,
		MaxListenerCount:             defaultMaxListenerCount,
		MaxConnectionCount:           defaultMaxConcurrentConnection,
		MaxServicePerClient:          defaultMaxServicePerClient,
		GreetingMessage:              defaultGreetingMessage,
	}
	configManager := NewServerConfigManager()
	configManager.UpdateConfigs(defaultConfig)
	container.Container.Singleton(func() IServerConfigManager {
		return configManager
	})
}
