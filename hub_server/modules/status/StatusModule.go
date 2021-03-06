package status

import (
	"encoding/json"
	"runtime"
	"time"
	"whub/common/async"
	"whub/common/ctimer"
	"whub/common/logger"
	"whub/common/observable"
	"whub/hub_server/context"
	"whub/hub_server/module_base"
)

const ID = "Status"
const DefaultStatReadInterval = time.Second * 30

type ServerStat struct {
	*SystemStat     `json:"systemStat"`
	AsyncPoolStat   *AsyncWorkerPoolStat `json:"asyncPoolStat"`
	ServicePoolStat *AsyncWorkerPoolStat `json:"servicePoolStat"`
}

type SystemStat struct {
	NumCpus       int    `json:"numCpus"`
	NumGoroutines int    `json:"numGoroutines"`
	Alloc         uint64 `json:"alloc"`
	TotalAlloc    uint64 `json:"totalAlloc"`
	SysAlloc      uint64 `json:"sysAlloc"`
	NumGC         uint32 `json:"numGC"`
}

func (s ServerStat) JsonByte() ([]byte, error) {
	return json.Marshal(s)
}

type AsyncWorkerPoolStat struct {
	Status            string `json:"status"`
	NumMaxWorkers     int    `json:"numMaxWorkers"`
	NumPendingTasks   int    `json:"numPendingTasks"`
	NumBusyWorkers    int    `json:"numBusyWorkers"`
	NumStartedWorkers int    `json:"numStartedWorkers"`
}

func (s AsyncWorkerPoolStat) JsonByte() ([]byte, error) {
	return json.Marshal(s)
}

type ServerStatusModule struct {
	*module_base.ModuleBase
	statReadTimer  ctimer.ICTimer
	observableStat observable.IObservable // observable with server stat
	asyncPool      async.IAsyncPool
	servicePool    async.IAsyncPool
	lastUpdateTime time.Time
	logger         *logger.SimpleLogger
}

type IServerStatusModule interface {
	GetServerStat() ServerStat
	SubscribeServerStatChange(cb func(stat ServerStat)) func()
}

func (m *ServerStatusModule) Init() error {
	m.ModuleBase = module_base.NewModuleBase(ID, nil)
	m.observableStat = observable.NewObservableWith(&ServerStat{
		SystemStat:      new(SystemStat),
		AsyncPoolStat:   new(AsyncWorkerPoolStat),
		ServicePoolStat: new(AsyncWorkerPoolStat),
	})
	m.asyncPool = context.Ctx.AsyncTaskPool()
	m.servicePool = context.Ctx.ServiceTaskPool()
	m.logger = m.Logger()
	m.statReadTimer = ctimer.New(DefaultStatReadInterval, m.readAndUpdateSystemStat)
	m.readAndUpdateSystemStat()
	m.statReadTimer.Repeat()
	return nil
}

func (c *ServerStatusModule) readSystemStat(stat *SystemStat) {
	var memStat runtime.MemStats
	stat.NumCpus = runtime.NumCPU()
	stat.NumGoroutines = runtime.NumGoroutine()
	runtime.ReadMemStats(&memStat)
	stat.Alloc = memStat.Alloc / 1024
	stat.TotalAlloc = memStat.TotalAlloc / 1024
	stat.SysAlloc = memStat.Sys / 1024
	stat.NumGC = memStat.NumGC
}

func (c *ServerStatusModule) readAsyncPoolStat(stat *AsyncWorkerPoolStat, asyncPool async.IAsyncPool) {
	stat.NumMaxWorkers = asyncPool.NumMaxWorkers()
	stat.NumBusyWorkers = asyncPool.NumBusyWorkers()
	stat.NumStartedWorkers = asyncPool.NumStartedWorkers()
	stat.NumPendingTasks = asyncPool.NumPendingTasks()
	stat.Status = asyncPool.Status()
}

func (c *ServerStatusModule) readAndUpdateSystemStat() {
	stat := c.observableStat.Get().(*ServerStat)
	c.readSystemStat(stat.SystemStat)
	c.readAsyncPoolStat(stat.AsyncPoolStat, c.asyncPool)
	c.readAsyncPoolStat(stat.ServicePoolStat, c.servicePool)

	c.observableStat.Set(stat)
	c.lastUpdateTime = time.Now()
	c.statReadTimer.Reset()
}

func (c *ServerStatusModule) GetServerStat() ServerStat {
	c.readAndUpdateSystemStat()
	return *(c.observableStat.Get().(*ServerStat))
}

func (c *ServerStatusModule) SubscribeServerStatChange(cb func(stat ServerStat)) func() {
	return c.observableStat.On(func(interface{}) {
		cb(c.GetServerStat())
	})
}
