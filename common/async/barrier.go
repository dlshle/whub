package async

import (
	"sync"
	"sync/atomic"
)

type Barrier struct {
	cond   *sync.Cond
	isOpen atomic.Value
}

func (b *Barrier) Open() {
	if !b.IsOpen() {
		b.isOpen.Store(true)
		b.cond.Broadcast()
	}
}

func (b *Barrier) Wait() {
	if b.IsOpen() {
		return
	}
	b.cond.L.Lock()
	b.cond.Wait()
	b.cond.L.Unlock()
}

func (b *Barrier) IsOpen() bool {
	return b.isOpen.Load().(bool)
}

func NewBarrier() *Barrier {
	b := &Barrier{
		sync.NewCond(&sync.Mutex{}),
		atomic.Value{},
	}
	b.isOpen.Store(false)
	return b
}

type StatefulBarrier struct {
	b     *Barrier
	state atomic.Value
}

func (s *StatefulBarrier) OpenWith(state interface{}) {
	if s.b.IsOpen() {
		return
	}
	s.state.Store(state)
	s.b.Open()
}

func (s *StatefulBarrier) Wait() {
	s.b.Wait()
}

func (s *StatefulBarrier) Get() interface{} {
	s.Wait()
	return s.state.Load()
}

func NewStatefulBarrier() *StatefulBarrier {
	return &StatefulBarrier{
		b:     NewBarrier(),
		state: atomic.Value{},
	}
}
