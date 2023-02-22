package simple_cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"simple_cache/mocks"
)

func Test_Storage(test *testing.T) {
	mockedTimeService := mocks.NewTimeService(test)
	mockedTimeService.On("GetCurrentTimestamp").Return(int64(1)).Once()
	mockedTimeService.On("GetCurrentTimestamp").Return(int64(2)).Once()
	mockedTimeService.On("GetCurrentTimestamp").Return(int64(3)).Once()
	mockedTimeService.On("GetCurrentTimestamp").Return(int64(4)).Once()

	testTTL := time.Second

	test.Run(
		"Create store with capacity for three elements. Set three elements and Get first of them. Then on setting fourth element displace one of not getting element",
		func(test *testing.T) {
			testTarget := NewService(
				context.TODO(),
				3,
				testTTL,
				testTTL,
				100*time.Millisecond,
				mockedTimeService,
			)

			testTarget.Set("foo", "1")
			testTarget.Set("bar", "2")
			testTarget.Set("test", "3")
			testTarget.Get("foo")
			testTarget.Get("foo")
			//waiting for reindex
			time.Sleep(2 * testTTL)
			testTarget.Set("clean", "4")

			assert.Equal(test, "1", testTarget.Get("foo"))
			assert.Equal(test, nil, testTarget.Get("bar"))
			assert.Equal(test, "3", testTarget.Get("test"))
			assert.Equal(test, "4", testTarget.Get("clean"))
		},
	)
}
