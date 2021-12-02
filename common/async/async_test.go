package async

import (
	"testing"
	"time"
	"whub/common/ctimer"
	"whub/common/test_utils"
)

func TestAsyncPool(t *testing.T) {
	pool := NewAsyncPool("test", 2, 5)
	pool.Verbose(true)
	test_utils.NewTestGroup("AsyncPool", "").Cases([]*test_utils.Assertion{
		test_utils.NewTestCase("basic scheduling", "", func() bool {
			b := NewStatefulBarrier()
			ctimer.New(time.Second, func() {
				b.OpenWith(false)
			}).Start()
			pool.Schedule(func() {
				b.OpenWith(true)
			})
			return b.Get().(bool)
		}),
	}).Do(t)
}
