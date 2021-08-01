package metering

import (
	"container/list"
	"time"
)

type markPair struct {
	time        time.Time
	description string
}

type StopWatch struct {
	id             string
	startTime      time.Time
	marks          *list.List
	onStopCallback func(*list.List)
}

type IStopWatch interface {
	Mark(description string)
	Stop()
}

func (w *StopWatch) Init(id string) {
	w.id = id
	w.marks = list.New()
	w.Mark("start")
}

func (w *StopWatch) Mark(description string) {
	w.marks.PushBack(&markPair{
		time:        time.Now(),
		description: description,
	})
}

func (w *StopWatch) Stop() {
	w.onStopCallback(w.marks)
	w.marks = nil
}
