package module_base

import (
	"fmt"
	"whub/common/logger"
	"whub/hub_server/context"
)

type IModule interface {
	Id() string
	Init() error
	OnLoad()
	Logger() *logger.SimpleLogger
	Dispose() error
	OnUnload()
}

type ModuleBase struct {
	id        string
	isLoaded  bool
	logger    *logger.SimpleLogger
	onDispose func() error
}

func NewModuleBase(id string, onDispose func() error) *ModuleBase {
	return &ModuleBase{
		id:        id,
		isLoaded:  false,
		logger:    context.Ctx.Logger().WithPrefix(fmt.Sprintf("[Module-%s]", id)),
		onDispose: onDispose,
	}
}

func (m *ModuleBase) Id() string {
	return m.id
}

func (m *ModuleBase) IsLoaded() bool {
	return m.isLoaded
}

func (m *ModuleBase) OnLoad() {
	m.logger.Printf("module %s is loaded", m.id)
}

func (m *ModuleBase) OnUnload() {
	m.logger.Printf("module %s is unloaded", m.id)
}

func (m *ModuleBase) Dispose() (err error) {
	if m.onDispose != nil {
		err = m.onDispose()
	}
	return
}

func (m *ModuleBase) Logger() *logger.SimpleLogger {
	return m.logger
}
