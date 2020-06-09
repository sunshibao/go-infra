package syncutil

import (
	"sync"
	"time"
)

/*
	谨慎使用，如果wg一直未返回，会导致go routine无法退出，有leak风险
 */
func WaitGroupTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan int, 1)
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return false
	case <-time.After(timeout):
		return true
	}
}
