package gomux

import (
	"bytes"
	"io"
	"sync"
	"testing"
)

func TestBroadcast(t *testing.T) {
	s := &Session{
		writers: map[io.WriteCloser]bool{},
	}

	broadcast := func(s *Session, c string) []*bytes.Buffer {
		r := []*bytes.Buffer{}
		ps := []io.ReadCloser{}
		var wg sync.WaitGroup
		// prepare 2 writers
		for _, _ = range []int{1, 2} {
			wg.Add(1)
			b := bytes.NewBuffer([]byte{})
			p, _ := s.StdoutPipe()
			// prepare the consumer
			go func() {
				defer wg.Done()
				io.Copy(b, p)
			}()
			r = append(r, b)
			ps = append(ps, p)
		}
		// broadcast c to all writers
		s.Write([]byte(c))
		for _, p := range ps {
			p.Close()
		}
		wg.Wait()
		return r
	}

	r := broadcast(s, "hello")
	for _, b := range r {
		if b.String() != "hello" {
			t.Fail()
		}
	}
}
