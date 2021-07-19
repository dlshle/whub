package async

import (
	"context"
	"fmt"
	"os"
	"sync"

	"wsdk/common/logger"
)

// TODO update pool size/worker size dynamically
// channel has better performance, so used Barrier to replace Promise

type AsyncError struct {
	msg string
}

func (e *AsyncError) Error() string {
	return e.msg
}

func NewAsyncError(msg string) error {
	return &AsyncError{msg}
}

type AsyncTask func()

type ComputableAsyncTask func() interface{}

const (
	IDLE        = 0
	RUNNING     = 1
	TERMINATING = 2
	TERMINATED  = 3
)

type AsyncPool struct {
	id            string
	context       context.Context
	cancelFunc    func()
	stopWaitGroup sync.WaitGroup
	rwLock        *sync.RWMutex
	channel       chan AsyncTask
	numWorkers    int
	numBusyWorker int
	status        int
	logger        *logger.SimpleLogger
}

type IAsyncPool interface {
	getStatus() int
	setStatus(status int)
	HasStarted() bool
	isRunning() bool
	Start()
	Stop()
	schedule(task AsyncTask)
	Schedule(task AsyncTask) *Barrier
	ScheduleComputable(computableTask ComputableAsyncTask) *StatefulBarrier
	Verbose(use bool)
}

func NewAsyncPool(id string, maxPoolSize, workerSize int) *AsyncPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &AsyncPool{
		id,
		ctx,
		cancel,
		sync.WaitGroup{},
		new(sync.RWMutex),
		make(chan AsyncTask, getInRangeInt(maxPoolSize, 16, 2048)),
		getInRangeInt(workerSize, 2, 1024),
		0,
		0,
		logger.New(os.Stdout, fmt.Sprintf("AsyncPool[pool-%s]", id), false),
	}
}

func (p *AsyncPool) withWrite(cb func()) {
	p.rwLock.Lock()
	defer p.rwLock.Unlock()
	cb()
}

func (p *AsyncPool) getStatus() int {
	p.rwLock.RLock()
	defer p.rwLock.RUnlock()
	return p.status
}

func (p *AsyncPool) setStatus(status int) {
	p.rwLock.Lock()
	defer p.rwLock.Unlock()
	if status > -1 && status < 4 {
		p.status = status
		p.logger.Printf("Pool status has transited to %d\n", status)
	}
	return
}

func (p *AsyncPool) HasStarted() bool {
	p.rwLock.RLock()
	defer p.rwLock.RUnlock()
	return p.status > IDLE
}

func (p *AsyncPool) isRunning() bool {
	p.rwLock.RLock()
	defer p.rwLock.RUnlock()
	return p.status == RUNNING
}

func (p *AsyncPool) incrementNumBusyWorkers() {
	p.withWrite(func() {
		p.numBusyWorker++
	})
}

func (p *AsyncPool) decrementNumBusyWorkers() {
	p.withWrite(func() {
		p.numBusyWorker--
	})
}

func (p *AsyncPool) Start() {
	if p.getStatus() > IDLE {
		return
	}
	go func() {
		// worker manager routine
		for i := 0; i < p.numWorkers; i++ {
			p.stopWaitGroup.Add(1)
			go func(wi int) {
				// worker routine
				for {
					select {
					case task, isOpen := <-p.channel:
						// simply take task and work on it sequentially
						if isOpen {
							p.logger.Printf("Worker %d has acquired task %p\n", wi, task)
							p.incrementNumBusyWorkers()
							task()
						} else {
							break
						}
						p.decrementNumBusyWorkers()
					case <-p.context.Done():
						break
					}
				}
				p.logger.Printf("Worker %d terminated\n", wi)
				p.stopWaitGroup.Done()
			}(i)
		}
		// wait till all workers terminated
		p.stopWaitGroup.Wait()
		p.setStatus(TERMINATED)
		p.logger.Printf("All worker has been terminated\n")
	}()
	p.setStatus(RUNNING)
}

func (p *AsyncPool) Stop() {
	if !p.HasStarted() {
		p.logger.Printf("Warn pool has not started\n")
		return
	}
	close(p.channel)
	p.cancelFunc()
	p.setStatus(TERMINATING)
	p.stopWaitGroup.Wait()
}

func (p *AsyncPool) schedule(task AsyncTask) {
	if !p.HasStarted() {
		p.Start()
	}
	p.channel <- task
	p.logger.Printf("Task %p has been scheduled\n", task)
}

// will block on channel buffer size exceeded
func (p *AsyncPool) Schedule(task AsyncTask) *Barrier {
	promise := NewBarrier()
	p.schedule(func() {
		task()
		promise.Open()
	})
	return promise
}

// will block on channel buffer size exceeded
func (p *AsyncPool) ScheduleComputable(computableTask ComputableAsyncTask) *StatefulBarrier {
	future := NewStatefulBarrier()
	p.schedule(func() {
		future.OpenWith(computableTask())
	})
	return future
}

func (p *AsyncPool) Verbose(use bool) {
	p.logger.Verbose(use)
}

// utils
func getInRangeInt(value, min, max int) int {
	if value < min {
		return min
	} else if value > max {
		return max
	} else {
		return value
	}
}
