package startable

import (
	"context"
	"errors"
	"sync"
	"time"
)

/*
========================
 Types
========================
*/

// PreStartCallback is called before start.
// Returning an error aborts startup.
type PreStartCallback func() error

// StartFunction is the main execution function.
// It should return when ctx is cancelled.
type StartFunction func(ctx context.Context) any

// PostStartCallback is called after StartFunction exits.
type PostStartCallback func(any)

type StartableState int

const (
	Idle StartableState = iota
	Starting
	Running
	Stopping
	Stopped
)

var (
	ErrAlreadyStarted = errors.New("already started")
	ErrNotStarted     = errors.New("not started")
	// ErrPreStartFailed = errors.New("pre-start failed")
	ErrStopTimeout    = errors.New("stop timeout exceeded")
)

/*
========================
 Startable
========================
*/

type Startable struct {
	mtx sync.Mutex

	ctx    context.Context
	cancel context.CancelFunc

	pre   PreStartCallback
	start StartFunction
	post  PostStartCallback

	state StartableState

	startSig chan struct{}
	endSig   chan struct{}
}

/*
========================
 Configuration
========================
*/

func (s *Startable) SetPreStartCallback(cb PreStartCallback) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if s.state != Idle && s.state != Stopped {
		return
	}
	s.pre = cb
}

func (s *Startable) SetStartFunction(fn StartFunction) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if s.state != Idle && s.state != Stopped {
		return
	}
	s.start = fn
}

func (s *Startable) SetPostStartCallback(cb PostStartCallback) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if s.state != Idle && s.state != Stopped {
		return
	}
	s.post = cb
}

/*
========================
 State helpers
========================
*/

func (s *Startable) State() StartableState {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	return s.state
}

// IsStarted reports whether the Startable is in running state or not.
func (s *Startable) IsRunning() bool {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	return s.state == Running
}
// IsActive reports whether the Startable is running sequence or not.
func (s *Startable) IsActive() bool {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if s.state != Idle && s.state != Stopped {
		return true
	}
	return false
}

/*
========================
 Lifecycle
========================
*/

func (s *Startable) Start(ctx context.Context) error {
	s.mtx.Lock()
	if s.state != Idle && s.state != Stopped {
		s.mtx.Unlock()
		return ErrAlreadyStarted
	}

	parent := ctx
	if parent == nil {
		parent = context.Background()
	}

	s.ctx, s.cancel = context.WithCancel(parent)
	s.startSig = make(chan struct{})
	s.endSig = make(chan struct{})
	s.state = Starting

	pre := s.pre
	start := s.start
	post := s.post

	s.mtx.Unlock()

	go func() {
		defer func() {
			s.mtx.Lock()
			s.state = Stopped
			close(s.endSig)
			s.mtx.Unlock()
		}()

		// Pre-start
		if pre != nil {
			if err := pre(); err != nil {
				close(s.startSig)
				return
			}
		}

		// Running
		s.mtx.Lock()
		s.state = Running
		s.mtx.Unlock()
		close(s.startSig)

		var out any
		if start != nil {
			out = start(s.ctx)
		}

		// Stopping
		s.mtx.Lock()
		s.state = Stopping
		s.mtx.Unlock()

		if post != nil {
			post(out)
		}
	}()

	return nil
}

/*
========================
 Synchronization
========================
*/
// WaitForFullStart blocks until the Startable has fully started.
// It returns true if the Startable started successfully, or false if it stopped before starting.
// if the Startable is not running, it returns false immediately.
// if the Startable is already started, it returns true immediately.
// if duration is non-zero, it will wait for that duration only.
func (s *Startable) WaitForFullStart(d time.Duration) bool {
	s.mtx.Lock()
	if s.state == Idle || s.state == Stopped {
		s.mtx.Unlock()
		return false
	}
	startSig := s.startSig
	s.mtx.Unlock()

	if d == 0 {
		<-startSig
		return s.IsRunning()
	}

	select {
	case <-startSig:
		return s.IsRunning()
	case <-time.After(d):
		return false
	}
}

func (s *Startable) Stop() error {
	s.mtx.Lock()
	if s.state == Idle || s.state == Stopped {
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

func (s *Startable) StopWithTimeout(d time.Duration) error {
	s.mtx.Lock()
	if s.state == Idle || s.state == Stopped {
		s.mtx.Unlock()
		return ErrNotStarted
	}
	cancel := s.cancel
	endSig := s.endSig
	s.mtx.Unlock()

	cancel()

	select {
	case <-endSig:
		return nil
	case <-time.After(d):
		return ErrStopTimeout
	}
}
