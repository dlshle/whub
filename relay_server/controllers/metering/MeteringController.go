package metering

import (
	"container/list"
	"fmt"
	"sync"
	"time"
	"wsdk/common/logger"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
)

type MeteringController struct {
	stopWatchPool *sync.Pool
	logger        *logger.SimpleLogger
}

type IMeteringController interface {
	Measure(id string) IStopWatch
}

func NewMeteringController() IMeteringController {
	return &MeteringController{
		stopWatchPool: &sync.Pool{
			New: func() interface{} {
				return &StopWatch{}
			},
		},
		logger: context.Ctx.Logger().WithPrefix("[MeteringController]"),
	}
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
	return stopWatch
}

func (c *MeteringController) stopWatchCallback(marks map[time.Time]string) {

}

func init() {
	container.Container.Singleton(func() IMeteringController {
		return NewMeteringController()
	})
}
