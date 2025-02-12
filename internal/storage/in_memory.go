package storage

import (
	"sync"
)

type SafeStringMap struct {
	m sync.Map
}

func NewSafeMap() *SafeStringMap {
	return &SafeStringMap{m: sync.Map{}}
}

func (sm *SafeStringMap) Store(key, value string) {
	sm.m.Store(key, value)
}

func (sm *SafeStringMap) Load(key string) (string, bool) {
	if val, ok := sm.m.Load(key); ok {
		if str, ok := val.(string); ok {
			return str, true
		}
	}
	return "", false
}

func (sm *SafeStringMap) Delete(key string) {
	sm.m.Delete(key)
}
