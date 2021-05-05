package WRCommon

import "sync"

const (
	QueueStatusIdle      = 0
	QueueStatusBuffering = 1
	QueueStatusOverload  = 2
)

type IBufferedQueue interface {
	Enqueue(interface{})
	Dequeue() interface{}
	Status() int
}

type BufferedChannelQueue struct {
	buffer chan interface{}
	status int
	lock   *sync.RWMutex
}

func (q *BufferedChannelQueue) withWrite(cb func()) {
	q.lock.Lock()
	defer q.lock.Unlock()
	cb()
}

func (q *BufferedChannelQueue) setStatus(status int) {
	q.withWrite(func() {
		q.status = status
	})
}

func (q *BufferedChannelQueue) Enqueue(data interface{}) {
	q.buffer <- data
	go func() {
		if len(q.buffer) >= cap(q.buffer) {
			q.setStatus(QueueStatusOverload)
		} else {
			q.setStatus(QueueStatusBuffering)
		}
	}()
}

func (q *BufferedChannelQueue) Dequeue() (data interface{}) {
	if len(q.buffer)-1 <= 0 {
		q.setStatus(QueueStatusIdle)
	}
	return <-q.buffer
}

func (q *BufferedChannelQueue) Status() int {
	q.lock.RLock()
	defer q.lock.RUnlock()
	return q.status
}
