package shutdown

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type Closer interface {
	Close() error
}

type ShutdownHandler struct {
	ctx      context.Context
	cancel   context.CancelFunc
	timeout  time.Duration
	closers  []namedCloser
	mu       sync.Mutex
	shutdown atomic.Bool
	once     sync.Once
	wg       sync.WaitGroup
}

type namedCloser struct {
	name   string
	closer Closer
}

func New(timeout time.Duration) *ShutdownHandler {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &ShutdownHandler{
		ctx:     ctx,
		cancel:  cancel,
		timeout: timeout,
	}
}

func (s *ShutdownHandler) Register(name string, closer Closer) {
	if closer == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.shutdown.Load() {
		return
	}

	s.closers = append(s.closers, namedCloser{name: name, closer: closer})
}

func (s *ShutdownHandler) RegisterFunc(name string, closeFn func() error) {
	if closeFn == nil {
		return
	}
	s.Register(name, funcCloser(closeFn))
}

type funcCloser func() error

func (f funcCloser) Close() error { return f() }

func (s *ShutdownHandler) Run() error {
	var runErr error

	s.once.Do(func() {
		s.shutdown.Store(true)

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		select {
		case sig := <-sigCh:
			runErr = s.doShutdown(fmt.Sprintf("received %s", sig))
		case <-s.ctx.Done():
			runErr = s.doShutdown("context cancelled")
		}

		signal.Stop(sigCh)
		close(sigCh)
	})

	return runErr
}

func (s *ShutdownHandler) doShutdown(reason string) error {
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), s.timeout)
	defer shutdownCancel()

	s.cancel()

	for i := len(s.closers) - 1; i >= 0; i-- {
		nc := s.closers[i]

		perResourceTimeout := s.timeout / 10
		if perResourceTimeout < time.Second {
			perResourceTimeout = time.Second
		}

		resourceCtx, resourceCancel := context.WithTimeout(shutdownCtx, perResourceTimeout)

		done := make(chan error, 1)
		s.wg.Add(1)

		go func(name string, c Closer) {
			defer s.wg.Done()
			defer resourceCancel()

			defer func() {
				if r := recover(); r != nil {
					done <- fmt.Errorf("panic while closing %s: %v", name, r)
				}
			}()

			done <- c.Close()
		}(nc.name, nc.closer)

		select {
		case err := <-done:
			if err != nil {
				_ = fmt.Errorf("failed to close %s: %w", nc.name, err)
			}
		case <-resourceCtx.Done():
			_ = fmt.Errorf("timeout closing %s", nc.name)
		case <-shutdownCtx.Done():
			_ = fmt.Errorf("shutdown timeout while closing %s", nc.name)
		}
	}

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-shutdownCtx.Done():
		return fmt.Errorf("shutdown timeout: some goroutines did not finish in time")
	}
}

func (s *ShutdownHandler) Context() context.Context {
	return s.ctx
}

func (s *ShutdownHandler) IsShuttingDown() bool {
	return s.shutdown.Load()
}

func (s *ShutdownHandler) Trigger() {
	s.once.Do(func() {
		s.shutdown.Store(true)
		s.cancel()
	})
}

func (s *ShutdownHandler) WaitForCompletion() {
	s.wg.Wait()
}
