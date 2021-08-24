package pubsub_v2

import "context"

// every message will first be put in store and then queue will fetch from store
// MessageQueue is a single coroutine that handles message dispatching
type IMessageQueue interface {
	Put(IPubSubMessage)
	Stop()
}

type MessageQueue struct {
	// dispatch queue
	q          chan IPubSubMessage
	ctx        context.Context
	cancelFunc func()

	lastMsgIndex uint32
}

func (q *MessageQueue) Put(m IPubSubMessage) {
	q.q <- m
}

func (q *MessageQueue) Stop() {
	q.cancelFunc()
}

// main goroutine of MessageQueue
func (q *MessageQueue) dispatcher() {

}
