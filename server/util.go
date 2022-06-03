package server

import "sync"

type item struct {
	value      string
	expireTime int64
}

type TTLMap struct {
	m map[string]*item
	l sync.Mutex
}
