package throttling

import (
	"errors"
	"fmt"
	"sync"
	"time"
	"wsdk/common/ctimer"
	"wsdk/common/logger"
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
	windowMap     *sync.Map
	cleanJobTimer ctimer.ICTimer
	logger        *logger.SimpleLogger
}

func NewThrottleController(logger *logger.SimpleLogger) IThrottleController {
	controller := &ThrottleController{
		windowMap: new(sync.Map),
		logger:    logger,
	}
	cleanJobTimer := ctimer.New(time.Minute, controller.cleanJob)
	controller.cleanJobTimer = cleanJobTimer
	cleanJobTimer.Repeat()
	return controller
}

func (c *ThrottleController) cleanJob() {
	now := time.Now()
	cleaned := 0
	c.windowMap.Range(func(key, value interface{}) bool {
		record := value.(*ThrottleRecord)
		if now.After(record.WindowExpiration) {
			c.windowMap.Delete(key)
			cleaned++
		}
		return true
	})
	c.logger.Printf("clean job done, %d records were removed", cleaned)
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
		return true
	})
}
