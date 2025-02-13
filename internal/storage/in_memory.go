package storage

import (
	"errors"
	"sync"
)

type SafeStringMap struct {
	m sync.Map
}

func NewSafeMap() *SafeStringMap {
	return &SafeStringMap{m: sync.Map{}}
}

func (sm *SafeStringMap) Store(key, value string) error {
	sm.m.Store(key, value)
	return nil
}

func (sm *SafeStringMap) Load(key string) (string, error) {
	if val, ok := sm.m.Load(key); ok {
		if str, ok := val.(string); ok {
			return str, nil
		}
	}
	return "", errors.New("value not found")
}

func (sm *SafeStringMap) Delete(key string) {
	sm.m.Delete(key)
}
