package metering

import (
	"container/list"
	"fmt"
	"sync"
	"time"
	"wsdk/common/logger"
)

type MeteringController struct {
	stopWatchPool *sync.Pool
	logger        *logger.SimpleLogger
	stopWatchMap  map[string]IStopWatch
	lock          *sync.RWMutex
}

type IMeteringController interface {
	Measure(id string) IStopWatch
	GetAssembledTraceId(prefix, id string) string
	Track(id string, description string) IStopWatch
	Stop(id string)
}

func NewMeteringController(logger *logger.SimpleLogger) IMeteringController {
	return &MeteringController{
		stopWatchPool: &sync.Pool{
			New: func() interface{} {
				return &StopWatch{}
			},
		},
		// logger: context.Ctx.Logger().WithPrefix("[MeteringController]"),
		logger:       logger,
		stopWatchMap: make(map[string]IStopWatch),
		lock:         new(sync.RWMutex),
	}
}

func (c *MeteringController) withWrite(cb func()) {
	c.lock.Lock()
	defer c.lock.Unlock()
	cb()
}

func (c *MeteringController) Measure(id string) IStopWatch {
	stopWatch := c.stopWatchPool.Get().(*StopWatch)
	stopWatch.Init(id)
	stopWatch.onStopCallback = func(marks *list.List) {
		var totalDiff time.Duration
		var lastTime time.Time
		lastDesc := ""
		for ele := marks.Front(); ele != nil; ele = ele.Next() {
			pair := ele.Value.(*markPair)
			if !lastTime.IsZero() {
				diff := pair.time.Sub(lastTime)
				totalDiff += diff
				c.logger.Println(fmt.Sprintf("[%s] diff between %s and %s are %s", id, lastDesc, pair.description, diff))
			}
			lastTime = pair.time
			lastDesc = pair.description
		}
		c.logger.Println(fmt.Sprintf("[%s] total time consumed: %s", id, totalDiff))
		c.stopWatchPool.Put(stopWatch)
	}
	c.withWrite(func() {
		c.stopWatchMap[id] = stopWatch
	})
	return stopWatch
}

func (c *MeteringController) GetAssembledTraceId(prefix, id string) string {
	return fmt.Sprintf("%s-%s", prefix, id)
}

func (c *MeteringController) Track(id string, description string) IStopWatch {
	c.lock.RLock()
	stopWatch := c.stopWatchMap[id]
	c.lock.RUnlock()
	if stopWatch == nil {
		// if no stopwatch is found, create new
		stopWatch = c.Measure(id)
	}
	stopWatch.Mark(description)
	return stopWatch
}

func (c *MeteringController) Stop(id string) {
	c.lock.RLock()
	stopWatch := c.stopWatchMap[id]
	c.lock.RUnlock()
	if stopWatch == nil {
		return
	}
	stopWatch.Stop()
	c.withWrite(func() {
		delete(c.stopWatchMap, id)
	})
}
