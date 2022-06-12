package server

import (
	"fmt"
	"testing"
	"time"
)

func TestTTLMap(t *testing.T) {
	ttlMap := NewTTLMap(10, int64(time.Millisecond*100))
	fmt.Println(ttlMap)
	ttlMap.Put("key", "value")
	fmt.Println(ttlMap.Get("key"))
	fmt.Println(ttlMap.Get("key not exist"))

	// fmt.Println("--> first gc")
	// runtime.GC()
	// time.Sleep(time.Millisecond * 200)
	// fmt.Println("--> second gc")
	// runtime.GC()
	// fmt.Println("--> third gc")
	// runtime.GC()
}
