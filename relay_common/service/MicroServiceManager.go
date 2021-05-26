package service

import (
	"sync"
	"wsdk/relay_common/messages"
)

type MicroServiceManager struct {
	serviceHandlers map[string]messages.MessageHandlerFunc
	lock *sync.RWMutex
}

type IMicroServiceManager interface {
	SupportsUri(uri string) bool
	Register(uri string, handler messages.MessageHandlerFunc) bool
	Unregister(uri string) bool
	GetHandler(uri string) messages.MessageHandlerFunc
}

func NewMicroServiceManager() IMicroServiceManager {
	return &MicroServiceManager{
		serviceHandlers: make(map[string]messages.MessageHandlerFunc),
		lock: new(sync.RWMutex),
	}
}

func (m *MicroServiceManager) withWrite(cb func()) {
	m.lock.Lock()
	defer m.lock.Unlock()
	cb()
}

func (m *MicroServiceManager) SupportsUri(shortUri string) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.GetHandler(shortUri) != nil
}

func (m *MicroServiceManager) Register(shortUri string, handler messages.MessageHandlerFunc) bool {
	if m.SupportsUri(shortUri) {
		return false
	}
	m.withWrite(func() {
		m.serviceHandlers[shortUri] = handler
	})
	return true
}

func (m *MicroServiceManager) Unregister(shortUri string) bool {
	if !m.SupportsUri(shortUri) {
		return false
	}
	m.withWrite(func() {
		delete(m.serviceHandlers, shortUri)
	})
	return true
}

func (m *MicroServiceManager) GetHandler(shortUri string) messages.MessageHandlerFunc {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.serviceHandlers[shortUri]
}