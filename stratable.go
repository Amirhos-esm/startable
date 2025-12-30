package startable

import (
	"context"
	"errors"
	"sync"
	"time"
)

// PreStartCallback is a function that is called before starting.
// If it returns false, the start process is aborted.
type PreStartCallback func() bool
type StartFunction func(ctx context.Context) any
type PostStartCallback func(any)

type Startable struct {
	mtx sync.Mutex

	ctx    context.Context
	cancel context.CancelFunc

	pre   PreStartCallback
	start StartFunction
	post  PostStartCallback

	state StartableState

	endSig   chan struct{}
	startSig chan struct{}
}

type StartableState int

const (
	Idle     StartableState = iota
	Starting                // pre + start not completed
	Running                 // start function executing
	Stopping
	Stopped
)

var (
	ErrAlreadyStarted = errors.New("already started")
	ErrNotStarted     = errors.New("not started")
	// ErrStopTimeout is returned when StopWithTimeout exceeds the given duration
	ErrStopTimeout = errors.New("stop timeout exceeded")
)

// SetPreStartCallback sets the pre-start callback.
func (s *Startable) SetPreStartCallback(cb PreStartCallback) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if !s.isIdle() {
		return
	}
	s.pre = cb
}

// SetStartFunction sets the start function.
func (s *Startable) SetStartFunction(fn StartFunction) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if !s.isIdle() {
		return
	}
	s.start = fn
}

// SetPostStartCallback sets the post-start callback.
func (s *Startable) SetPostStartCallback(cb PostStartCallback) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if !s.isIdle() {
		return
	}
	s.post = cb
}

// IsActive reports whether the Startable is running sequence or not.
func (s *Startable) IsActive() bool {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	return !s.isIdle()
}

// IsRunning reports whether the Startable is on Running state or not.
func (s *Startable) IsRunning() bool {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	return s.state == Running
}

func (s *Startable) State() StartableState {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	return s.state
}

func (s *Startable) isIdle() bool {
	return s.state == Idle || s.state == Stopped
}

// Start initializes and runs the Startable.
// if ctx is nil it will use background context
func (s *Startable) Start(ctx context.Context) error {
	s.mtx.Lock()
	if !s.isIdle() {
		s.mtx.Unlock()
		return ErrAlreadyStarted
	}

	if ctx == nil {
		ctx = context.Background()
	}
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.endSig = make(chan struct{})
	s.startSig = make(chan struct{})
	s.state = Starting

	pre := s.pre
	post := s.post
	start := s.start

	var out any

	s.mtx.Unlock()

	go func() {
		// Run pre-start
		if pre != nil {
			if pre() == false {
				close(s.startSig)
				goto end
			}
		}

		s.mtx.Lock()
		s.state = Running
		s.mtx.Unlock()

		close(s.startSig)
		// Run main start function

		if start != nil {
			out = start(s.ctx)
		}

		s.mtx.Lock()
		s.state = Stopping
		s.mtx.Unlock()

		// Run post-start
		if post != nil {
			post(out)
		}

	end:
		// Mark as stopped
		s.mtx.Lock()
		s.state = Stopped
		close(s.endSig)
		s.mtx.Unlock()
	}()

	return nil
}

// WaitForFullStart blocks until the Startable has fully started.
// It returns true if the Startable started successfully, or false if it stopped before starting.
// if the Startable is not running, it returns false immediately.
// if the Startable is already started, it returns true immediately.
// if duration is non-zero, it will wait for that duration only.
func (s *Startable) WaitForFullStart(duration time.Duration) bool {
	s.mtx.Lock()
	if s.isIdle() {
		s.mtx.Unlock()
		return false
	}
	s.mtx.Unlock()

	if duration == 0 {
		<-s.startSig
		return s.IsRunning()

	}
	select {
	case <-s.startSig:
		// started
		return s.IsRunning()
	case <-time.After(duration):
		return false
	}
}

// Stop cancels the running Startable.
func (s *Startable) Stop() error {
	s.mtx.Lock()
	if s.isIdle() {
		s.mtx.Unlock()
		return ErrNotStarted
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
	if s.isIdle() {
		s.mtx.Unlock()
		return ErrNotStarted
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
