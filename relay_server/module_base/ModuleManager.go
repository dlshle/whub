package module_base

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"unsafe"
)

var Manager IModuleManager

func init() {
	Manager = NewModuleManager()
}

const TagModule = "module"

type IModuleManager interface {
	// cuz we're gonna register uninitialized modules,
	RegisterModule(module IModule) error
	RegisterModules(modules []IModule) error
	UnregisterModule(id string) error
	Clear()
	GetModule(id string) IModule
	AutoFill(object interface{}) error
}

type ModuleManager struct {
	modules map[string]IModule
	lock    *sync.RWMutex
}

func NewModuleManager() IModuleManager {
	return ModuleManager{
		modules: make(map[string]IModule),
		lock:    new(sync.RWMutex),
	}
}

func (m ModuleManager) withWrite(cb func()) {
	m.lock.Lock()
	defer m.lock.Unlock()
	cb()
}

// only register with hollow instance(new(moduleX))
func (m ModuleManager) RegisterModule(module IModule) error {
	err := module.Init()
	if err != nil {
		return err
	}
	if m.GetModule(module.Id()) != nil {
		return errors.New(fmt.Sprintf("module [%s] has already been registered", module.Id()))
	}
	m.withWrite(func() {
		m.modules[module.Id()] = module
	})
	return nil
}

// atomic register
func (m ModuleManager) RegisterModules(modules []IModule) (err error) {
	var numRegistered int
	for _, module := range modules {
		if err = m.RegisterModule(module); err != nil {
			break
		}
		numRegistered++
	}
	if err != nil {
		for i := 0; i < numRegistered; i++ {
			m.UnregisterModule(modules[i].Id())
		}
	}
	return
}

func (m ModuleManager) UnregisterModule(id string) error {
	module := m.GetModule(id)
	if module == nil {
		return errors.New(fmt.Sprintf("module [%s] does not exist", module.Id()))
	}
	return m.unregisterModule(module)
}

func (m ModuleManager) unregisterModule(module IModule) (err error) {
	err = module.Dispose()
	m.withWrite(func() {
		delete(m.modules, module.Id())
	})
	return
}

func (m ModuleManager) Clear() {
	for k, _ := range m.modules {
		m.UnregisterModule(k)
	}
}

func (m ModuleManager) GetModule(id string) IModule {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.modules[id]
}

func (m ModuleManager) AutoFill(object interface{}) error {
	if object == nil {
		return nil
	}
	return m.autoFill(object)
}

func (m ModuleManager) autoFill(object interface{}) error {
	recvType := reflect.TypeOf(object)
	if recvType.Kind() == reflect.Ptr {
		return m.autoFillValue(reflect.ValueOf(object).Elem())
	} else if recvType.Kind() == reflect.Struct {
		return m.autoFillValue(reflect.ValueOf(object))
	}
	return errors.New("invalid object type for AutoFill")
}

func (m ModuleManager) autoFillValue(value reflect.Value) error {
	for i := 0; i < value.NumField(); i++ {
		f := value.Field(i)
		if t, ok := value.Type().Field(i).Tag.Lookup(TagModule); ok {
			if len(t) == 0 {
				return errors.New("empty module tag identifier")
			}
			module := m.GetModule(t)
			if module == nil {
				return errors.New(fmt.Sprintf("can not find module by id %s", t))
			}
			ptr := reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem()
			if !ptr.CanSet() {
				return errors.New(fmt.Sprintf("can not set field %s with module %s", f.String(), module.Id()))
			}
			ptr.Set(reflect.ValueOf(module))
		}
	}
	return nil
}
