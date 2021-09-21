package throttling

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type ThrottleRecord struct {
	Id               string
	WindowExpiration time.Time
	WindowDuration   time.Duration
	HitsUnderWindow  int
	Limit            int
}

func NewThrottleRecord(id string, limit int, duration time.Duration) *ThrottleRecord {
	return &ThrottleRecord{
		Id:               id,
		WindowExpiration: time.Now().Add(duration),
		WindowDuration:   duration,
		HitsUnderWindow:  1,
		Limit:            limit,
	}
}

func (r *ThrottleRecord) restartBy(windowBegin time.Time) {
	r.WindowExpiration = windowBegin.Add(r.WindowDuration)
	if r.HitsUnderWindow > r.Limit {
		r.HitsUnderWindow = r.HitsUnderWindow - r.Limit
		exceededHits := r.HitsUnderWindow - r.Limit
		// penalty on exceeded hits are added to the next throttling window
		r.WindowExpiration = r.WindowExpiration.Add(time.Minute * time.Duration(exceededHits))
	}
	r.HitsUnderWindow = 0
}

func (r *ThrottleRecord) Hit() (ThrottleRecord, error) {
	now := time.Now()
	if r.WindowExpiration.Before(now) {
		r.restartBy(now)
	}
	r.HitsUnderWindow++
	if r.HitsUnderWindow >= r.Limit {
		return *r, errors.New(fmt.Sprintf(
			"number of requests exceeded throttle limit in window by %d hits, next window begins at %s, penalty will be added to the next window",
			r.HitsUnderWindow-r.Limit,
			r.WindowExpiration.String()))
	}
	return *r, nil
}

type IThrottleController interface {
	Hit(id string, limit int, duration time.Duration) (ThrottleRecord, error)
	Clear()
}

type ThrottleController struct {
	// TODO add a job to periodically clean the map so that we don't leave too much "dead" data here
	windowMap *sync.Map
}

func NewThrottleController() IThrottleController {
	return &ThrottleController{
		windowMap: new(sync.Map),
	}
}

func (c *ThrottleController) Hit(id string, limit int, duration time.Duration) (ThrottleRecord, error) {
	record := c.createOrLoadRecord(id, limit, duration)
	return record.Hit()
}

func (c *ThrottleController) createOrLoadRecord(id string, limit int, duration time.Duration) *ThrottleRecord {
	record, _ := c.windowMap.LoadOrStore(id, NewThrottleRecord(id, limit, duration))
	return record.(*ThrottleRecord)
}

func (c *ThrottleController) Clear() {
	c.windowMap.Range(func(key, value interface{}) bool {
		c.windowMap.Delete(key)
		return false
	})
}
