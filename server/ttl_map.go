package server

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var tLog = logrus.New().WithField("package", "ttl_map")

type item struct {
	value      interface{}
	lastAccess int64
}

type TTLMap struct {
	m   map[string]*item
	mux *sync.RWMutex
}

func NewTTLMap(ln int, maxTTL int64) (m *TTLMap) {
	m = &TTLMap{m: make(map[string]*item, ln), mux: new(sync.RWMutex)}
	tLog.Info("create new ttl map")
	go func() {
		for now := range time.Tick(10 * time.Second) {
			func() {
				tLog.Info("refresh new ttl map", m.m)
				m.mux.Lock()
				defer m.mux.Unlock()
				for k, v := range m.m {
					if now.Unix()-v.lastAccess > int64(maxTTL) {
						delete(m.m, k)
					}
				}
			}()
		}
	}()
	return
}

func (m *TTLMap) Len() int {
	return len(m.m)
}

func (m *TTLMap) Put(k string, v interface{}) {
	m.mux.Lock()
	defer m.mux.Unlock()
	it, ok := m.m[k]
	if !ok {
		it = &item{value: v}
		m.m[k] = it
	}
	it.lastAccess = time.Now().Unix()
}

func (m *TTLMap) Get(k string) (v interface{}) {
	m.mux.RLock()
	defer m.mux.RUnlock()
	if it, ok := m.m[k]; ok {
		v = it.value
		it.lastAccess = time.Now().Unix()
	}
	return
}
