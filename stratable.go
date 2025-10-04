package startable

import (
	"context"
	"errors"
	"sync"
	"time"
)

type PreStartCallback func()
type StartFunction func(ctx context.Context) any
type PostStartCallback func(any)

type Startable struct {
	mtx       sync.Mutex
	isStarted bool

	ctx    context.Context
	cancel context.CancelFunc

	pre   PreStartCallback
	start StartFunction
	post  PostStartCallback

	endSig chan struct{}
}

var (
	ErrAlreadyStarted = errors.New("already started")
	ErrNothingToStop  = errors.New("nothing to stop")
	// ErrStopTimeout is returned when StopWithTimeout exceeds the given duration
	ErrStopTimeout = errors.New("stop timeout exceeded")
)

// SetPreStartCallback sets the pre-start callback.
func (s *Startable) SetPreStartCallback(cb PreStartCallback) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if s.isStarted {
		return
	}
	s.pre = cb
}

// SetStartFunction sets the start function.
func (s *Startable) SetStartFunction(fn StartFunction) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if s.isStarted {
		return
	}
	s.start = fn
}

// SetPostStartCallback sets the post-start callback.
func (s *Startable) SetPostStartCallback(cb PostStartCallback) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if s.isStarted {
		return
	}
	s.post = cb
}

// IsStarted reports whether the Startable is running.
func (s *Startable) IsStarted() bool {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	return s.isStarted
}

// Start initializes and runs the Startable.
func (s *Startable) Start() error {
	s.mtx.Lock()
	if s.isStarted {
		s.mtx.Unlock()
		return ErrAlreadyStarted
	}

	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.endSig = make(chan struct{})
	s.isStarted = true
	s.mtx.Unlock()

	go func() {
		// Run pre-start
		if s.pre != nil {
			s.pre()
		}

		// Run main start function
		var out any
		if s.start != nil {
			out = s.start(s.ctx)
		}

		// Run post-start
		if s.post != nil {
			s.post(out)
		}

		// Mark as stopped
		s.mtx.Lock()
		s.isStarted = false
		close(s.endSig)
		s.mtx.Unlock()
	}()

	return nil
}

// Stop cancels the running Startable.
func (s *Startable) Stop() error {
	s.mtx.Lock()
	if !s.isStarted {
		s.mtx.Unlock()
		return ErrNothingToStop
	}
	cancel := s.cancel
	endSig := s.endSig
	s.mtx.Unlock()

	cancel()
	<-endSig
	return nil
}

// StopWithTimeout attempts to stop the Startable, but returns an error if it doesn't stop within d.
func (s *Startable) StopWithTimeout(d time.Duration) error {
	s.mtx.Lock()
	if !s.isStarted {
		s.mtx.Unlock()
		return ErrNothingToStop
	}
	cancel := s.cancel
	endSig := s.endSig
	s.mtx.Unlock()

	// Cancel the context
	cancel()

	// Wait for either endSig or timeout
	select {
	case <-endSig:
		return nil
	case <-time.After(d):
		return ErrStopTimeout
	}
}
