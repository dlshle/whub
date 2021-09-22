package blocklist

import (
	"time"
)

// TODO

type IBlockListController interface {
	Add(record string, ttl time.Duration) error
	Has(record string) (bool, error)
	Remove(record string) error
}

type BlockListController struct {
	store IBlockListStore
}
