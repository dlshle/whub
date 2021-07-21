package controllers

import (
	"encoding/json"
	"runtime"
	"time"
	"wsdk/common/ctimer"
	"wsdk/common/logger"
	"wsdk/common/observable"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
)

const DefaultStatReadInterval = time.Second * 30

type SystemStat struct {
	NumCpus       int              `json:"numCpus"`
	NumGoroutines int              `json:"numGoroutines"`
	MemStat       runtime.MemStats `json:"memStat"`
}

func (s SystemStat) JsonByte() ([]byte, error) {
	return json.Marshal(s)
}

// TODO get pool stat here as well
type AsyncPoolStat struct {
	Status          string `json:"status"`
	NumWorkers      int    `json:"numWorkers"`
	NumPendingTasks int    `json:"numPendingTasks"`
	NumBusyWorkers  int    `json:"numBusyWorkers"`
}

func (s AsyncPoolStat) JsonByte() ([]byte, error) {
	return json.Marshal(s)
}

type SystemStatusController struct {
	statReadTimer  ctimer.ICTimer
	observableStat observable.IObservable // observable with system stat
	lastUpdateTime time.Time
	logger         *logger.SimpleLogger
}

type ISystemStatusController interface {
	GetSystemStat() SystemStat
	SubscribeSystemStatChange(cb func(stat SystemStat)) func()
}

func NewSystemStatusController() ISystemStatusController {
	controller := &SystemStatusController{
		observableStat: observable.NewObservableWith(new(SystemStat)),
		logger:         context.Ctx.Logger().WithPrefix("[SystemStatusController]"),
	}
	controller.readAndUpdateSystemStat()
	controller.statReadTimer = ctimer.New(DefaultStatReadInterval, controller.readAndUpdateSystemStat)
	controller.statReadTimer.Repeat()
	return controller
}

func (c *SystemStatusController) readAndUpdateSystemStat() {
	stat := c.observableStat.Get().(*SystemStat)
	stat.NumCpus = runtime.NumCPU()
	stat.NumGoroutines = runtime.NumGoroutine()
	runtime.ReadMemStats(&stat.MemStat)
	c.observableStat.Set(stat)
	c.lastUpdateTime = time.Now()
	jsonByte, _ := stat.JsonByte()
	c.logger.Println("system status has been updated: ", (string)(jsonByte))
}

func (c *SystemStatusController) GetSystemStat() SystemStat {
	return *(c.observableStat.Get().(*SystemStat))
}

func (c *SystemStatusController) SubscribeSystemStatChange(cb func(stat SystemStat)) func() {
	return c.observableStat.On(func(interface{}) {
		cb(c.GetSystemStat())
	})
}

func init() {
	container.Container.Singleton(func() ISystemStatusController {
		return NewSystemStatusController()
	})
}
