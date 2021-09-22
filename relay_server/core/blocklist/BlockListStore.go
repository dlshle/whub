package blocklist

import (
	"errors"
	"sync"
	"time"
	"wsdk/common/ctimer"
)

type IBlockListStore interface {
	Add(record string, ttl time.Duration) (bool, error)
	Has(record string) (bool, error)
	Delete(record string) (bool, error)
}

type ttlRecord struct {
	record   string
	expireAt time.Time
}

type InMemoryBlockListStore struct {
	records       map[string]*ttlRecord
	lock          *sync.RWMutex
	cleanJobTimer ctimer.ICTimer
}

func (s *InMemoryBlockListStore) cleanJob() {
	var expiredRecordIds []string
	now := time.Now()
	s.lock.RLock()
	for k, v := range s.records {
		if v.expireAt.After(now) {
			expiredRecordIds = append(expiredRecordIds, k)
		}
	}
	s.lock.RUnlock()
	for _, id := range expiredRecordIds {
		s.Delete(id)
	}
}

func (s *InMemoryBlockListStore) withWrite(cb func()) {
	s.lock.Lock()
	defer s.lock.Unlock()
	cb()
}

func (s *InMemoryBlockListStore) Add(record string, ttl time.Duration) (exist bool, err error) {
	now := time.Now()
	s.withWrite(func() {
		if oldRecord := s.records[record]; oldRecord != nil {
			exist = true
			if oldRecord.expireAt.After(now) {
				return
			}
			delete(s.records, record)
		}
		s.records[record] = &ttlRecord{
			record:   record,
			expireAt: now.Add(ttl),
		}
	})
	return
}

func (s *InMemoryBlockListStore) Has(id string) (exist bool, err error) {
	s.lock.RLock()
	record := s.records[id]
	s.lock.RUnlock()
	exist = false
	if record == nil {
		return
	} else if record.expireAt.Before(time.Now()) {
		s.Delete(id)
		return
	} else {
		exist = true
	}
	return
}

func (s *InMemoryBlockListStore) Delete(id string) (success bool, err error) {
	success = false
	s.withWrite(func() {
		if s.records[id] == nil {
			err = errors.New("record does not exist")
			return
		}
		delete(s.records, id)
		success = true
	})
	return
}

func NewInMemoryBlockListStore() IBlockListStore {
	store := &InMemoryBlockListStore{
		records: make(map[string]*ttlRecord),
		lock:    new(sync.RWMutex),
	}
	timer := ctimer.New(time.Minute*time.Duration(30), store.cleanJob)
	timer.Repeat()
	return store
}
