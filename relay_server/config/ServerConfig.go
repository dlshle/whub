package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
)

var Config ServerConfig

type ServerConfig struct {
	CommonConfig     `json:"commonConfig"`
	DomainConfigs    `json:"domainConfig"`
	DisabledServices []string `json:"disabledServices"`
}

type CommonConfig struct {
	MaxAsyncPoolSize             int    `json:"maxAsyncPoolSize"`
	MaxServiceAsyncPoolSize      int    `json:"maxServiceAsyncPoolSize"`
	AsyncPoolWorkerFactor        int    `json:"asyncPoolWorkerFactor"`
	ServiceAsyncPoolWorkerFactor int    `json:"serviceAsyncPoolWorkerFactor"`
	MaxListenerCount             int    `json:"maxListenerCount"`
	MaxConnectionCount           int    `json:"maxConnectionCount"`
	MaxServicePerClient          int    `json:"maxServicePerClient"`
	SignKey                      string `json:"signKey"`
}

const (
	defaultMaxListenerCount        = 512
	defaultAsyncPoolSize           = 2048
	defaultServicePoolSize         = 1024
	defaultAsyncPoolWorkerFactor   = 32
	defaultServicePoolWorkerFactor = 16
	defaultMaxConcurrentConnection = 2048
	defaultSignKey                 = "d1s7218U7!d-r5b"
)

type DomainConfigs map[string]DomainConfig

type DomainConfig struct {
	Persistent PersistentConfig `json:"persistent"`
	Redis      RedisConfig      `json:"redis"`
}

type PersistentConfig struct {
	Driver   string `json:"driver"`
	Server   string `json:"server"`
	Username string `json:"username"`
	Password string `json:"password"`
	Db       string `json:"db"`
}

type RedisConfig struct {
	Server   string `json:"server"`
	Password string `json:"password"`
}

func init() {
	var configPath string
	Config.CommonConfig = CommonConfig{
		MaxListenerCount:             defaultMaxListenerCount,
		MaxAsyncPoolSize:             defaultAsyncPoolSize,
		MaxServiceAsyncPoolSize:      defaultServicePoolSize,
		AsyncPoolWorkerFactor:        defaultAsyncPoolWorkerFactor,
		ServiceAsyncPoolWorkerFactor: defaultServicePoolWorkerFactor,
		MaxConnectionCount:           defaultMaxConcurrentConnection,
		SignKey:                      defaultSignKey,
	}
	flag.StringVar(&configPath, "config", "", "path to the server config json file")
	flag.Parse()
	if configPath == "" {
		fmt.Println("no config path is specified, will use default config")
		return
	}
	configStream, err := readServerConfig(configPath)
	if err != nil {
		fmt.Printf("unable to read config file from %s due to %s, will use default config\n", configPath, err.Error())
		return
	}
	config, err := parseServerConfig(configStream)
	if err != nil {
		fmt.Printf("unable to parse config file from %s due to %s, will use default config\n", configPath, err.Error())
		return
	}
	printParsedServerConfig(config)
	Config = config
}

func readServerConfig(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(f)
}

func printParsedServerConfig(config ServerConfig) {
	marshalled, err := json.Marshal(config)
	if err != nil {
		panic(err)
	}
	fmt.Printf("server config: %s\n", (string)(marshalled))
}

func parseServerConfig(configStream []byte) (serverConfig ServerConfig, err error) {
	err = json.Unmarshal(configStream, &serverConfig)
	return
}
