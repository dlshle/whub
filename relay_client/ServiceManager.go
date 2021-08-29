package relay_client

import (
	"errors"
	"fmt"
	"sync"
	"wsdk/common/logger"
	"wsdk/common/uri_trie"
	"wsdk/relay_client/clients"
	"wsdk/relay_client/connections"
	"wsdk/relay_client/container"
	"wsdk/relay_client/context"
	"wsdk/relay_common/connection"
)

type IServiceManager interface {
	GetServiceById(id string) (svc IClientService)
	RegisterService(service IClientService) (err error)
	UnregisterService(service IClientService) (err error)
	UnregisterAllServices() (err error)
	MatchServiceByUri(uri string) (ctx *uri_trie.MatchContext, err error)
}

type ServiceManager struct {
	trie               *uri_trie.TrieTree
	services           map[string]IClientService
	lock               *sync.RWMutex
	pool               connections.IConnectionPool `$inject:""`
	serviceConnections []connection.IConnection
	unfitConnChan      chan connection.IConnection
	relayServiceClient clients.IRelayServiceClient
	logger             *logger.SimpleLogger
}

func NewServiceManager(primaryConn connection.IConnection) IServiceManager {
	manager := &ServiceManager{
		trie:               uri_trie.NewTrieTree(),
		services:           make(map[string]IClientService),
		lock:               new(sync.RWMutex),
		unfitConnChan:      make(chan connection.IConnection, context.Ctx.MaxActiveServiceConnections()),
		relayServiceClient: clients.NewRelayServiceClient(context.Ctx.Identity().Id(), primaryConn),
		logger:             context.Ctx.Logger().WithPrefix("[ServiceManager]"),
	}
	err := container.Container.Fill(manager)
	if err != nil {
		panic(err)
	}
	container.Container.Singleton(func() IServiceManager {
		return manager
	})
	err = manager.initServiceConnections()
	if err != nil {
		panic(err)
	}
	context.Ctx.AsyncTaskPool().Schedule(manager.maintainConnectionWorker)
	return manager
}

func (m *ServiceManager) withWrite(cb func()) {
	m.lock.Lock()
	defer m.lock.Unlock()
	cb()
}

func (m *ServiceManager) withRead(cb func()) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	cb()
}

func (m *ServiceManager) maintainConnectionWorker() {
	for {
		select {
		case <-context.Ctx.Context().Done():
			m.UnregisterAllServices()
			m.logger.Println("context done received, stop maintain job")
			return
		case conn := <-m.unfitConnChan:
			m.pool.Put(conn)
			err := m.produceConnection()
			if err != nil {
				m.logger.Println("produce connection failed, stop maintain job")
				return
			}
		}
	}
}

func (m *ServiceManager) produceConnection() error {
	conn, err := m.pool.Get()
	m.logger.Println("new service connection produced")
	if err != nil {
		return err
	}
	conn.OnError(func(error) {
		conn.Close()
	})
	conn.OnClose(func(error) {
		m.unfitConnChan <- conn
	})
	m.withRead(func() {
		for _, svc := range m.services {
			err = m.relayServiceClient.UpdateServiceProvider(conn, svc.Describe())
			if err != nil {
				m.logger.Printf(fmt.Sprintf("failed to update service provider from new connection for service %s due to %s", svc.Id(), err.Error()))
			} else {
				m.logger.Printf(fmt.Sprintf("update service provider from new connection for service %s succedded", svc.Id()))
			}
		}
	})
	m.withWrite(func() {
		for i := range m.serviceConnections {
			c := m.serviceConnections[i]
			if c == nil || !c.IsLive() {
				m.serviceConnections[i] = conn
				return
			}
		}
	})
	return nil
}

func (m *ServiceManager) updateServiceProviders(service IClientService) {
	m.withRead(func() {
		for _, conn := range m.serviceConnections {
			m.relayServiceClient.UpdateServiceProvider(conn, service.Describe())
		}
	})
}

func (m *ServiceManager) initServiceConnections() (err error) {
	maxActiveCount := context.Ctx.MaxActiveServiceConnections()
	m.serviceConnections = make([]connection.IConnection, maxActiveCount, maxActiveCount)
	for i := 0; i < maxActiveCount; i++ {
		err = m.produceConnection()
		if err != nil {
			return
		}
	}
	return
}

func (m *ServiceManager) RegisterService(service IClientService) (err error) {
	err = service.Init(context.Ctx.Server())
	if err != nil {
		return err
	}
	m.withRead(func() {
		if m.services[service.Id()] != nil {
			err = errors.New("service already registered")
		}
	})
	if err != nil {
		return err
	}
	m.withWrite(func() {
		m.services[service.Id()] = service
		shortUris := service.ServiceUris()
		for i, uri := range shortUris {
			err = m.trie.Add(uri, service, true)
			if err != nil {
				for j := i; j > -1; j-- {
					m.trie.Remove(shortUris[j])
				}
				return
			}
		}
	})
	if err != nil {
		return
	}
	err = service.Register()
	if err != nil {
		return
	}
	m.updateServiceProviders(service)
	return nil
}

func (m *ServiceManager) UnregisterService(service IClientService) (err error) {
	m.withRead(func() {
		if m.services[service.Id()] == nil {
			err = errors.New("service does not exist")
		}
	})
	if err != nil {
		return
	}
	service.Stop()
	m.withWrite(func() {
		for _, uri := range service.ServiceUris() {
			m.trie.Remove(uri)
		}
		delete(m.services, service.Id())
	})
	return
}

func (m *ServiceManager) UnregisterAllServices() (err error) {
	m.withWrite(func() {
		for _, svc := range m.services {
			svc.Stop()
			for _, uri := range svc.ServiceUris() {
				m.trie.Remove(uri)
			}
			delete(m.services, svc.Id())
		}
	})
	return
}

func (m *ServiceManager) GetServiceById(id string) (svc IClientService) {
	m.withRead(func() {
		svc = m.services[id]
	})
	return
}

func (m *ServiceManager) MatchServiceByUri(uri string) (ctx *uri_trie.MatchContext, err error) {
	m.withRead(func() {
		ctx, err = m.trie.Match(uri)
	})
	return
}
