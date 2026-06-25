package service

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestRunDeviceExtractExclusive_blocksConcurrent(t *testing.T) {
	s := NewStorageService(nil, nil, nil, nil)
	deviceID := "test-device-lock"
	var running int32

	release := make(chan struct{})
	go func() {
		_ = s.RunDeviceExtractExclusive(deviceID, func() error {
			atomic.StoreInt32(&running, 1)
			<-release
			atomic.StoreInt32(&running, 0)
			return nil
		})
	}()

	waitUntil(t, func() bool { return atomic.LoadInt32(&running) == 1 }, 500*time.Millisecond)

	entered := make(chan struct{})
	go func() {
		_ = s.RunDeviceExtractExclusive(deviceID, func() error {
			close(entered)
			return nil
		})
	}()

	select {
	case <-entered:
		t.Fatal("second job should not start while first holds lock")
	case <-time.After(80 * time.Millisecond):
	}

	close(release)
	<-entered
}

func waitUntil(t *testing.T, cond func() bool, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}
