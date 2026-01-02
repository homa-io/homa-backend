// Package drivers provides the integration driver registry and common types.
package drivers

import (
	"sync"
)

var (
	registry     = make(map[string]Driver)
	registryLock sync.RWMutex
)

// Register adds a driver to the registry.
func Register(driver Driver) {
	registryLock.Lock()
	defer registryLock.Unlock()
	registry[driver.Type()] = driver
}

// Get returns a driver by type.
func Get(typeID string) (Driver, bool) {
	registryLock.RLock()
	defer registryLock.RUnlock()
	driver, ok := registry[typeID]
	return driver, ok
}

// GetAll returns all registered drivers.
func GetAll() []Driver {
	registryLock.RLock()
	defer registryLock.RUnlock()

	result := make([]Driver, 0, len(registry))
	for _, driver := range registry {
		result = append(result, driver)
	}
	return result
}

// AllTypes returns all registered type IDs.
func AllTypes() []string {
	registryLock.RLock()
	defer registryLock.RUnlock()

	result := make([]string, 0, len(registry))
	for typeID := range registry {
		result = append(result, typeID)
	}
	return result
}
