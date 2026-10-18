package linkid

import (
	"sync"

	"github.com/koss-null/list"
)

// LinkID 链接ID池
// 链接ID从1开始，最大3000
// 链接ID用完后，从1开始循环使用

type LinkID struct {
	m sync.Mutex
	l list.Linked[int]
}

var pool = &LinkID{}

func init() {
	var i int
	for i = 1; i <= 1000; i++ {
		pool.l.PushBack(i)
	}
}

func Get() int {
	pool.m.Lock()
	defer pool.m.Unlock()

	id, ok := pool.l.PopFront()
	if !ok {
		return 0
	}
	return id
}

func Release(id int) {
	pool.m.Lock()
	defer pool.m.Unlock()

	pool.l.PushBack(id)
}
