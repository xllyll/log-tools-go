package service

import (
	"sync"
)

// RunDeviceExtractExclusive runs fn while holding the per-device extract/import lock.
// Concurrent callers for the same deviceID block until the current operation finishes.
func (s *StorageService) RunDeviceExtractExclusive(deviceID string, fn func() error) error {
	lock := s.deviceExtractLock(deviceID)
	lock.Lock()
	defer lock.Unlock()
	return fn()
}

func (s *StorageService) deviceExtractLock(deviceID string) *sync.Mutex {
	key := sanitizeDeviceID(deviceID)
	v, _ := s.extractMu.LoadOrStore(key, &sync.Mutex{})
	return v.(*sync.Mutex)
}
