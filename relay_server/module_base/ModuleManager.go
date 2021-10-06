package module_base

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"unsafe"
	"wsdk/common/logger"
	"wsdk/relay_server/context"
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
	modules       map[string]IModule
	moduleTypeMap map[reflect.Type]IModule
	lock          *sync.RWMutex
	logger        *logger.SimpleLogger
}

func NewModuleManager() IModuleManager {
	return ModuleManager{
		modules:       make(map[string]IModule),
		moduleTypeMap: make(map[reflect.Type]IModule),
		lock:          new(sync.RWMutex),
		logger:        context.Ctx.Logger().WithPrefix("[ModuleManager]"),
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
	reflectType := reflect.TypeOf(module)
	m.withWrite(func() {
		m.modules[module.Id()] = module
		m.moduleTypeMap[reflectType] = module
	})
	m.logger.Printf("module %s is registered", module.Id())
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
	m.logger.Printf("module %s is unregistered", module.Id())
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

func (m ModuleManager) autoFill(object interface{}) (err error) {
	defer func() {
		if err != nil {
			m.logger.Printf("autofill failed due to %s", err.Error())
		}
	}()
	recvType := reflect.TypeOf(object)
	if recvType.Kind() == reflect.Ptr {
		err = m.autoFillValue(reflect.ValueOf(object).Elem())
		return
	} else if recvType.Kind() == reflect.Struct {
		err = m.autoFillValue(reflect.ValueOf(object))
		return
	}
	err = errors.New("invalid object type for AutoFill")
	return
}

func (m ModuleManager) autoFillValue(value reflect.Value) (err error) {
	for i := 0; i < value.NumField(); i++ {
		f := value.Field(i)
		if t, ok := value.Type().Field(i).Tag.Lookup(TagModule); ok {
			if len(t) == 0 {
				err = m.autoFillByType(f)
			} else {
				err = m.autoFillById(f, t)
			}
			if err != nil {
				return
			}
		}
	}
	return nil
}

func (m ModuleManager) autoFillById(f reflect.Value, id string) (err error) {
	module := m.GetModule(id)
	if module == nil {
		return errors.New(fmt.Sprintf("can not find module by id %s", id))
	}
	err = m.tryToFillField(f, module)
	return
}

func (m ModuleManager) autoFillByType(reflectValue reflect.Value) error {
	var module IModule
	reflectType := reflectValue.Type()
	m.lock.RLock()
	module = m.moduleTypeMap[reflectType]
	// only search when miss
	if module == nil {
		for k, v := range m.moduleTypeMap {
			if k.Implements(reflectType) {
				module = v
			}
		}
	}
	m.lock.RUnlock()
	if module == nil {
		return errors.New(fmt.Sprintf("can not find module by type %s", reflectType.String()))
	}
	return m.tryToFillField(reflectValue, module)
}

func (m ModuleManager) tryToFillField(f reflect.Value, module IModule) error {
	ptr := reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem()
	if !ptr.CanSet() {
		return errors.New(fmt.Sprintf("can not set field %s with module %s", f.String(), module.Id()))
	}
	ptr.Set(reflect.ValueOf(module))
	return nil
}
