package async

import (
	"context"
	"fmt"
	"os"
	"sync"

	"wsdk/common/logger"
)

const (
	MaxOutPolicyWait = 0 // wait for next available worker
	maxOutPolicyRun  = 1 // run on new goroutine
)

var statusStringMap map[byte]string

func init() {
	statusStringMap = make(map[byte]string)
	statusStringMap[IDLE] = "IDLE"
	statusStringMap[RUNNING] = "RUNNING"
	statusStringMap[TERMINATING] = "TERMINATING"
	statusStringMap[TERMINATED] = "TERMINATED"
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
	id                string
	context           context.Context
	cancelFunc        func()
	stopWaitGroup     sync.WaitGroup
	rwLock            *sync.RWMutex
	channel           chan AsyncTask
	numTotWorkers     int
	numStartedWorkers int
	numBusyWorkers    int
	status            byte
	logger            *logger.SimpleLogger
	maxPoolSize       int
	maxOutPolicy      uint8
}

type IAsyncPool interface {
	getStatus() byte
	setStatus(status byte)
	HasStarted() bool
	start()
	Stop()
	schedule(task AsyncTask)
	Schedule(task AsyncTask) *WaitLock
	ScheduleComputable(computableTask ComputableAsyncTask) *StatefulBarrier
	Verbose(use bool)
	NumMaxWorkers() int
	NumStartedWorkers() int
	NumPendingTasks() int
	NumBusyWorkers() int
	Status() string
	IncreaseWorkerSizeTo(size int) bool
}

func NewAsyncPool(id string, maxPoolSize, workerSize int) IAsyncPool {
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
		0,
		logger.New(os.Stdout, fmt.Sprintf("AsyncPool[%s]", id), false),
		maxPoolSize,
		MaxOutPolicyWait,
	}
}

func (p *AsyncPool) withWrite(cb func()) {
	p.rwLock.Lock()
	defer p.rwLock.Unlock()
	cb()
}

func (p *AsyncPool) getStatus() byte {
	p.rwLock.RLock()
	defer p.rwLock.RUnlock()
	return p.status
}

func (p *AsyncPool) setStatus(status byte) {
	p.rwLock.Lock()
	defer p.rwLock.Unlock()
	if status >= 0 && status < 4 {
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

func (p *AsyncPool) incrementNumBusyWorkers() {
	p.withWrite(func() {
		p.numBusyWorkers++
	})
}

func (p *AsyncPool) decrementNumBusyWorkers() {
	p.withWrite(func() {
		p.numBusyWorkers--
	})
}

func (p *AsyncPool) runWorker(index int) {
	// worker routine
	for {
		select {
		case task, isOpen := <-p.channel:
			// simply take task and work on it sequentially
			if isOpen {
				p.logger.Printf("Worker %d has acquired task %p\n", index, task)
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
	p.logger.Printf("Worker %d terminated\n", index)
	p.stopWaitGroup.Done()
}

func (p *AsyncPool) tryAddAndRunWorker() {
	if p.getStatus() > RUNNING {
		p.logger.Println("status is terminating or terminated, can not add new worker")
		return
	}
	p.withWrite(func() {
		if p.numStartedWorkers < p.numTotWorkers {
			// need to increment the waitGroup before worker goroutine runs
			p.stopWaitGroup.Add(1)
			// of course worker runs on its own goroutine
			go p.runWorker(p.numStartedWorkers)
			p.logger.Printf("worker %d has been started", p.numStartedWorkers)
			p.numStartedWorkers++
		}
	})
}

func (p *AsyncPool) initWorker() {
	p.tryAddAndRunWorker()
	p.logger.Println("first worker has been initiated")
	// wait till all workers terminated
	p.stopWaitGroup.Wait()
	p.setStatus(TERMINATED)
	p.logger.Println("All worker has been terminated")
}

func (p *AsyncPool) start() {
	if p.getStatus() > IDLE {
		return
	}
	p.setStatus(RUNNING)
	go p.initWorker()
}

func (p *AsyncPool) Stop() {
	if !p.HasStarted() {
		p.logger.Println("Warn pool has not started")
		return
	}
	close(p.channel)
	p.cancelFunc()
	p.setStatus(TERMINATING)
	p.stopWaitGroup.Wait()
}

func (p *AsyncPool) schedule(task AsyncTask) {
	status := p.getStatus()
	switch {
	case status == IDLE:
		p.start()
	case status > RUNNING:
		return
	}
	// if all currently running workers are busy, try to add a new worker
	if p.NumBusyWorkers() == p.NumStartedWorkers() {
		p.tryAddAndRunWorker()
	}
	// TODO: schedule the task or run task immediately depending on maxOutPolicy
	p.channel <- task
	p.logger.Printf("Task %p has been scheduled\n", task)
}

// will block on channel buffer size exceeded
func (p *AsyncPool) Schedule(task AsyncTask) *WaitLock {
	promise := NewWaitLock()
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

func (p *AsyncPool) NumMaxWorkers() int {
	p.rwLock.RLock()
	defer p.rwLock.RUnlock()
	return p.numTotWorkers
}

func (p *AsyncPool) NumPendingTasks() int {
	if p.getStatus() == RUNNING {
		return len(p.channel)
	}
	return 0
}

func (p *AsyncPool) NumBusyWorkers() int {
	if p.getStatus() == RUNNING {
		p.rwLock.RLock()
		defer p.rwLock.RUnlock()
		return p.numBusyWorkers
	}
	return 0
}

func (p *AsyncPool) NumStartedWorkers() int {
	p.rwLock.RLock()
	defer p.rwLock.RUnlock()
	return p.numStartedWorkers
}

func (p *AsyncPool) Status() string {
	return statusStringMap[p.getStatus()]
}

func (p *AsyncPool) IncreaseWorkerSizeTo(size int) bool {
	if size > p.NumMaxWorkers() {
		p.withWrite(func() {
			p.numTotWorkers = size
		})
		return true
	}
	return false
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
