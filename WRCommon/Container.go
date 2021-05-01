package WRCommon

import "sync"

type ServiceContainer struct {
	m    map[string]IService
	lock *sync.RWMutex
}

func (c *ServiceContainer) withWrite(cb func()) {
	c.lock.Lock()
	defer c.lock.Unlock()
	cb()
}

func (c *ServiceContainer) Has(id string) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.m[id] != nil
}

func (c *ServiceContainer) Add(service IService) bool {
	if c.Has(service.Id()) {
		return false
	}
	c.withWrite(func() {
		c.m[service.Id()] = service
	})
	return true
}

func (c *ServiceContainer) Remove(id string) bool {
	if c.Has(id) {
		return false
	}
	c.withWrite(func() {
		delete(c.m, id)
	})
	return true
}

func (c *ServiceContainer) GetService(id string) IService {
	if !c.Has(id) {
		return nil
	}
	return c.m[id]
}
