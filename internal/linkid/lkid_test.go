package linkid

import (
	"sync"
	"testing"
	"time"
)

func TestLinkID(t *testing.T) {
	wg := sync.WaitGroup{}
	for range 10 {
		wg.Go(func() {
			id := Get()
			defer Release(id)

			// time.Sleep(time.Millisecond * 100)
			t.Logf("linkid: %d", id)
		})

		time.Sleep(time.Millisecond * 10)
	}

	wg.Wait()
}
