package blocklist

import (
	"time"
	"wsdk/common/logger"
	"wsdk/relay_server/container"
	"wsdk/relay_server/context"
)

const (
	DefaultBlockListTtl = time.Hour
)

type IBlockListController interface {
	DemoteByAddr(addr string) error
	Add(record string, ttl time.Duration) error
	Has(record string) (bool, error)
	Remove(record string) error
}

type BlockListController struct {
	store  IBlockListStore
	logger *logger.SimpleLogger
}

func NewBlockListController() IBlockListController {
	return BlockListController{
		store:  NewInMemoryBlockListStore(),
		logger: context.Ctx.Logger().WithPrefix("[BlockListController]"),
	}
}

func (c BlockListController) Add(record string, ttl time.Duration) error {
	_, err := c.store.Add(record, ttl)
	if err != nil {
		c.logger.Printf("record %s is not added to the store due to %s", record, err.Error())
	} else {
		c.logger.Printf("record %s was added to the store", record)
	}
	return err
}

func (c BlockListController) DemoteByAddr(addr string) error {
	return c.Add(addr, DefaultBlockListTtl)
}

func (c BlockListController) Has(record string) (bool, error) {
	return c.store.Has(record)
}

func (c BlockListController) Remove(record string) error {
	err := c.store.Delete(record)
	if err != nil {
		c.logger.Printf("record %s is not removed from the store due to %s", record, err.Error())
	} else {
		c.logger.Printf("record %s has been removed from the store", record)
	}
	return err
}

func Load() error {
	return container.Container.Singleton(func() IBlockListController {
		return NewBlockListController()
	})
}
