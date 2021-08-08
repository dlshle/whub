package data_structures

import "sync"

type ISet interface {
	Add(interface{}) bool
	Delete(interface{}) bool
	GetAll() []interface{}
	Clear()
	Size() int
}

type Set struct {
	m map[interface{}]bool
}

func NewSet() ISet {
	return &Set{make(map[interface{}]bool)}
}

func (s *Set) Add(data interface{}) bool {
	if s.m[data] {
		return false
	}
	s.m[data] = true
	return true
}

func (s *Set) Delete(data interface{}) bool {
	if s.m[data] {
		delete(s.m, data)
		return true
	}
	return false
}

func (s *Set) Clear() {
	for k := range s.m {
		delete(s.m, k)
	}
}

func (s *Set) GetAll() []interface{} {
	var data []interface{}
	for k, _ := range s.m {
		data = append(data, k)
	}
	return data
}

func (s *Set) Size() int {
	return len(s.m)
}

type SafeSet struct {
	lock *sync.RWMutex
	s    ISet
}

func (s *SafeSet) withWrite(cb func()) {
	s.lock.Lock()
	defer s.lock.Unlock()
	cb()
}

func (s *SafeSet) Add(i interface{}) (exist bool) {
	s.withWrite(func() {
		exist = s.s.Add(i)
	})
	return
}

func (s *SafeSet) Delete(i interface{}) (exist bool) {
	s.withWrite(func() {
		exist = s.s.Delete(i)
	})
	return
}

func (s *SafeSet) Clear() {
	s.withWrite(func() {
		s.s.Clear()
	})
}

func (s *SafeSet) GetAll() []interface{} {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.s.GetAll()
}

func (s *SafeSet) Size() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.s.Size()
}

func NewSafeSet() ISet {
	return &SafeSet{
		lock: new(sync.RWMutex),
		s:    NewSet(),
	}
}
