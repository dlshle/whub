package blocklist

import (
	"time"
	"whub/common/logger"
	"whub/hub_server/module_base"
	"whub/hub_server/modules/middleware_manager"
)

const (
	ID                  = "BlockList"
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

func (c *BlockListModule) Init() error {
	c.ModuleBase = module_base.NewModuleBase(ID, nil)
	c.store = NewInMemoryBlockListStore()
	c.logger = c.Logger()
	return nil
}

func (c *BlockListModule) OnLoad() {
	if err := middleware_manager.RegisterMiddleware(new(BlockListMiddleware)); err != nil {
		c.Logger().Printf("unable to load block list middleware due to %s", err.Error())
	}
	c.ModuleBase.OnLoad()
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
