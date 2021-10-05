package client_store

import (
	"time"
	"wsdk/relay_server/client"
)

type DClientQuery struct {
	limit         int
	createdAfter  time.Time
	createdBefore time.Time
	cType         int
}

func (q *DClientQuery) Limit(n int) *DClientQuery {
	q.limit = n
	return q
}

func (q *DClientQuery) CreatedWithin(after time.Time, before time.Time) *DClientQuery {
	if after.Before(before) {
		q.createdAfter = after
		q.createdBefore = before
	}
	return q
}

func (q *DClientQuery) CreatedAfter(after time.Time) *DClientQuery {
	if !q.createdBefore.IsZero() && q.createdBefore.Before(after) {
		return q
	}
	q.createdAfter = after
	return q
}

func (q *DClientQuery) CreatedBefore(before time.Time) *DClientQuery {
	if !q.createdAfter.IsZero() && q.createdAfter.After(before) {
		return q
	}
	q.createdBefore = before
	return q
}

func (q *DClientQuery) Type(t int) *DClientQuery {
	q.cType = t
	return q
}

func Query() *DClientQuery {
	return &DClientQuery{cType: -1}
}

type IClientStore interface {
	Get(id string) (*client.Client, error)
	GetAll() ([]*client.Client, error)
	Create(client *client.Client) error
	Update(client *client.Client) error
	Has(id string) (bool, error)
	Delete(id string) error
	Find(query *DClientQuery) ([]*client.Client, error)
	Close() error
}
