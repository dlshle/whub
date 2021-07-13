package config

import "wsdk/relay_server/container"

const (
	ServerConfigId = "ServerConfig"

	defaultMaxListenerCount        = 1024
	defaultAsyncPoolSize           = 2048
	defaultServicePoolSize         = 1024
	defaultAsyncPoolWorkerFactor   = 32
	defaultServicePoolWorkerFactor = 16
	defaultMaxConcurrentConnection = 2048
	defaultMaxServicePerClient     = 8
	defaultGreetingMessage         = ""
)

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

func init() {
	defaultConfig := &ServerConfig{
		MaxAsyncPoolSize:             defaultAsyncPoolSize,
		MaxServiceAsyncPoolSize:      defaultServicePoolSize,
		AsyncPoolWorkerFactor:        defaultAsyncPoolWorkerFactor,
		ServiceAsyncPoolWorkerFactor: defaultServicePoolWorkerFactor,
		MaxListenerCount:             defaultMaxListenerCount,
		MaxConnectionCount:           defaultMaxConcurrentConnection,
		MaxServicePerClient:          defaultMaxServicePerClient,
		GreetingMessage:              defaultGreetingMessage,
	}

	container.Container.RegisterComponent(ServerConfigId, defaultConfig)
}
