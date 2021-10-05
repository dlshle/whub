package blocklist

import (
	"time"
	"wsdk/common/logger"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
	"wsdk/relay_server/module_base"
)

const (
	DefaultBlockListTtl = time.Hour
)

type IBlockListModule interface {
	DemoteByAddr(addr string) error
	Add(record string, ttl time.Duration) error
	Has(record string) (bool, error)
	Remove(record string) error
}

type BlockListModule struct {
	*module_base.ModuleBase
	store  IBlockListStore
	logger *logger.SimpleLogger
}

func NewBlockListModule() IBlockListModule {
	return &BlockListModule{
		store:  NewInMemoryBlockListStore(),
		logger: context.Ctx.Logger().WithPrefix("[BlockListModule]"),
	}
}

func (c *BlockListModule) Init() error {
	c.ModuleBase = module_base.NewModuleBase("BlockList", func() error {
		var holder IBlockListModule
		return container.Container.RemoveByType(holder)
	})
	c.store = NewInMemoryBlockListStore()
	c.logger = c.Logger()
	return container.Container.Singleton(func() IBlockListModule {
		return c
	})
}

func (c *BlockListModule) Add(record string, ttl time.Duration) error {
	_, err := c.store.Add(record, ttl)
	if err != nil {
		c.logger.Printf("record %s is not added to the store due to %s", record, err.Error())
	} else {
		c.logger.Printf("record %s was added to the store", record)
	}
	return err
}

func (c *BlockListModule) DemoteByAddr(addr string) error {
	return c.Add(addr, DefaultBlockListTtl)
}

func (c *BlockListModule) Has(record string) (bool, error) {
	return c.store.Has(record)
}

func (c *BlockListModule) Remove(record string) error {
	err := c.store.Delete(record)
	if err != nil {
		c.logger.Printf("record %s is not removed from the store due to %s", record, err.Error())
	} else {
		c.logger.Printf("record %s has been removed from the store", record)
	}
	return err
}

func Load() error {
	return container.Container.Singleton(func() IBlockListModule {
		return NewBlockListModule()
	})
}
