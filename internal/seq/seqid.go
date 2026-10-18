package seq

// you need to close the done channel to stop the generator
func NewSeqIdGenerator(start uint32) (<-chan uint32, chan<- struct{}) {
	out := make(chan uint32, 64)
	done := make(chan struct{})

	go func() {
		var i uint32 = start
		for {
			select {
			case out <- i:
				i++
			case <-done:
				close(out)
				return
			}
		}
	}()
	return out, done
}
