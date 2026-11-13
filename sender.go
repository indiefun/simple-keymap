package main

import "sync"

// SendInput can wait for global hooks to finish. Never call it from a hook:
// the hook must be able to return while the sender is waiting. This worker owns
// all output and preserves key-down/key-up order without blocking the UI thread.
type inputSender struct {
	mu      sync.Mutex
	pending []output
	closing bool
	wake    chan struct{}
	done    chan struct{}
	deliver func([]output) bool
	failed  func()
}

func newInputSender(deliver func([]output) bool, failed func()) *inputSender {
	s := &inputSender{
		wake: make(chan struct{}, 1), done: make(chan struct{}),
		deliver: deliver, failed: failed,
	}
	go s.run()
	return s
}

func (s *inputSender) enqueue(out []output) {
	if len(out) == 0 {
		return
	}
	s.mu.Lock()
	if !s.closing {
		s.pending = append(s.pending, out...)
	}
	s.mu.Unlock()
	s.signal()
}

func (s *inputSender) signal() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// close drops queued actions and releases delivered holds. Waiting is separate so
// closing a window never waits for SendInput on the hook-owning UI thread.
func (s *inputSender) close() {
	s.mu.Lock()
	s.closing = true
	// Do not replay queued clicks after the user closes the window. The
	// currently in-flight output is followed only by releases for held keys.
	s.pending = nil
	s.mu.Unlock()
	s.signal()
}

func (s *inputSender) run() {
	defer close(s.done)
	pressed := map[string]bool{}
	for {
		<-s.wake
		for {
			s.mu.Lock()
			batch, closing := s.pending, s.closing
			s.pending = nil
			s.mu.Unlock()
			if len(batch) == 0 {
				if closing {
					// Also release successful outputs on shutdown if the engine
					// did not get a matching physical key-up before exit.
					var releases []output
					for id := range pressed {
						releases = append(releases, output{Key: id})
					}
					if len(releases) != 0 {
						s.deliver(releases)
					}
					return
				}
				break
			}
			if !s.deliver(batch) {
				// The helper owns recovery for partial native delivery. No more
				// requests may be sent through a failed transport.
				s.mu.Lock()
				s.closing = true
				s.pending = nil
				s.mu.Unlock()
				if s.failed != nil {
					s.failed()
				}
				return
			}
			for _, o := range batch {
				if o.Down {
					pressed[o.Key] = true
				} else {
					delete(pressed, o.Key)
				}
			}
		}
	}
}
