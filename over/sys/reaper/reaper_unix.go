package reaper

import (
	"errors"
	"os/exec"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

// ErrNoSuchProcess is returned when the process no longer exists
var ErrNoSuchProcess = errors.New("no such process")

const bufferSize = 32

type subscriber struct {
	sync.Mutex
	c      chan Exit
	closed bool
}

func (s *subscriber) close() {
	s.Lock()
	if s.closed {
		s.Unlock()
		return
	}
	close(s.c)
	s.closed = true
	s.Unlock()
}

func (s *subscriber) do(fn func()) {
	s.Lock()
	fn()
	s.Unlock()
}

type Exit struct {
	Timestamp time.Time
	Pid       int
	Status    int
}

// Default is the default monitor initialized for the package
var Default = &monitor{
	subscribers: make(map[chan Exit]*subscriber),
}

// monitor monitors the underlying system for process status changes
type monitor struct {
	sync.Mutex
	subscribers map[chan Exit]*subscriber
}

// Wait blocks until a process is signal as dead.
// User should rely on the value of the exit status to determine if the
// command was successful or not.
func (m *monitor) Wait(c *exec.Cmd, ec chan Exit) (int, error) {
	for e := range ec {
		if e.Pid == c.Process.Pid {
			// make sure we flush all IO
			c.Wait()
			m.Unsubscribe(ec)
			return e.Status, nil
		}
	}
	// return no such process if the ec channel is closed and no more exit
	// events will be sent
	return -1, ErrNoSuchProcess
}

func (m *monitor) getSubscribers() map[chan Exit]*subscriber {
	out := make(map[chan Exit]*subscriber)
	m.Lock()
	for k, v := range m.subscribers {
		out[k] = v
	}
	m.Unlock()
	return out
}

func (m *monitor) notify(e Exit) chan struct{} {
	const timeout = 1 * time.Millisecond
	var (
		done    = make(chan struct{}, 1)
		timer   = time.NewTimer(timeout)
		success = make(map[chan Exit]struct{})
	)
	stop(timer, true)

	go func() {
		defer close(done)

		for {
			var (
				failed      int
				subscribers = m.getSubscribers()
			)
			for _, s := range subscribers {
				s.do(func() {
					if s.closed {
						return
					}
					if _, ok := success[s.c]; ok {
						return
					}
					timer.Reset(timeout)
					recv := true
					select {
					case s.c <- e:
						success[s.c] = struct{}{}
					case <-timer.C:
						recv = false
						failed++
					}
					stop(timer, recv)
				})
			}
			// all subscribers received the message
			if failed == 0 {
				return
			}
		}
	}()
	return done
}

func stop(timer *time.Timer, recv bool) {
	if !timer.Stop() && recv {
		<-timer.C
	}
}

// exit is the wait4 information from an exited process
type exit struct {
	Pid    int
	Status int
}

// Reap 获取调用进程的所有子进程并返回它们的退出信息
func reap(wait bool) (exits []exit, err error) {
	var (
		ws  unix.WaitStatus
		rus unix.Rusage
	)
	flag := unix.WNOHANG
	if wait {
		flag = 0
	}
	for {
		pid, err := unix.Wait4(-1, &ws, flag, &rus)
		if err != nil {
			if err == unix.ECHILD {
				return exits, nil
			}
			return exits, err
		}
		if pid <= 0 {
			return exits, nil
		}
		exits = append(exits, exit{
			Pid:    pid,
			Status: exitStatus(ws),
		})
	}
}

const exitSignalOffset = 128

// exitStatus returns the correct exit status for a process based on if it
// was signaled or exited cleanly
func exitStatus(status unix.WaitStatus) int {
	if status.Signaled() {
		return exitSignalOffset + int(status.Signal())
	}
	return status.ExitStatus()
}

// Subscribe to process exit changes
func (m *monitor) Subscribe() chan Exit {
	c := make(chan Exit, bufferSize)
	m.Lock()
	m.subscribers[c] = &subscriber{
		c: c,
	}
	m.Unlock()
	return c
}

func (m *monitor) Unsubscribe(c chan Exit) {
	m.Lock()
	s, ok := m.subscribers[c]
	if !ok {
		m.Unlock()
		return
	}
	s.close()
	delete(m.subscribers, c)
	m.Unlock()
}

// Start starts the command and registers the process with the reaper
func (m *monitor) Start(c *exec.Cmd) (chan Exit, error) {
	ec := m.Subscribe()
	if err := c.Start(); err != nil {
		m.Unsubscribe(ec)
		return nil, err
	}
	return ec, nil
}
func Reap() error {
	now := time.Now()
	exits, err := reap(false)
	for _, e := range exits {
		done := Default.notify(Exit{
			Timestamp: now,
			Pid:       e.Pid,
			Status:    e.Status,
		})

		select {
		case <-done:
		case <-time.After(1 * time.Second):
		}
	}
	return err
}
