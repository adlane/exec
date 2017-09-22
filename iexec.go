package exec

import (
	"bytes"
	"io"
	"os/exec"
	"time"
)

const (
	minRead = 4096
)

// InteractiveExec executes a process and keep it running in background, while providing functions to write to and read from process ios (stdin, stdout, stderr).
// The program path is defined by the name arguments, args are passed as arguments to the program.
//
// TimeoutedExec returns process execution context.
func InteractiveExec(name string, args ...string) ProcessContext {
	res := &processContext{}
	res.proc = exec.Command(name, args...)
	res.stdout = newStdio()
	res.stderr = newStdio()
	res.stdin = newStdio()
	res.cancel = make(chan struct{}, 1)

	res.proc.Stdout = res.stdout
	res.proc.Stderr = res.stderr
	res.proc.Stdin = res.stdin

	go res.proc.Run()

	return res
}

// Process context interface defines how to interact with a running process in the background
type ProcessContext interface {
	// Read from the running process output streams (stdout and stderr)
	Receive(stdReader Reader, timeout time.Duration)

	// Stop the running process and release resources
	Stop() error

	// Cancel current receive
	Cancel()

	// Write to the running process input stream (stdin)
	Send(s string) error
}

// Reader interface provides a list of callbacks that will be called during a ProcessContext.Receive call.
type Reader interface {
	// Called when new data is ready to read on the running process stdout stream
	OnData(b []byte) bool

	// Called when new data is ready to read on the running process stderr stream
	OnError(b []byte) bool

	// Called on ProcessContext.Receive timeout
	OnTimeout()
}

type processContext struct {
	proc   *exec.Cmd
	stdin  *stdio
	stdout *stdio
	stderr *stdio
	cancel chan struct{}
}

// Receive data from the running process (stdout, stderr)
func (p *processContext) Receive(stdReader Reader, timeout time.Duration) {
	t := time.Now()
	deadline := t.Add(timeout)

	stop := false
	for {
		select {
		case <-time.After(timeout):
			if stdReader != nil {
				stdReader.OnTimeout()
			}
			stop = true
		case <-p.stdout.ready:
			if b := p.stdout.readAll(); len(b) > 0 {
				if stdReader != nil {
					stop = stdReader.OnData(b)
				}
			}
		case <-p.stderr.ready:
			if b := p.stderr.readAll(); len(b) > 0 {
				if stdReader != nil {
					stop = stdReader.OnError(b)
				}
			}
		case <-p.cancel:
			stop = true
		}

		t = time.Now()
		if t.After(deadline) {
			stop = true
		} else {
			timeout = deadline.Sub(t)
		}

		if stop {
			break
		}
	}
}

// Stop process and release resources
func (p *processContext) Stop() error {
	return p.proc.Process.Kill()
}

// Cancel current receive
func (p *processContext) Cancel() {
	p.cancel <- struct{}{}
}

// Send text to the running process (stdin)
func (p *processContext) Send(s string) error {
	if _, err := p.stdin.buf.Write([]byte(s)); err != nil {
		return err
	} else {
		p.stdin.ready <- err
	}
	return nil
}

type stdio struct {
	buf   bytes.Buffer
	ready chan error
}

func newStdio() *stdio {
	res := &stdio{}
	res.buf.Grow(minRead)
	res.ready = make(chan error, 10)
	return res
}

func (p *stdio) readAll() []byte {
	res := []byte{}
	for b := p.buf.Next(minRead); len(b) > 0; b = p.buf.Next(minRead) {
		res = append(res, b...)
	}
	return res
}

// stdout, stderr
func (p *stdio) ReadFrom(r io.Reader) (n int64, err error) {
	b := make([]byte, minRead)
	readSize := 0
	for {
		readSize, err = r.Read(b)
		n += int64(readSize)
		if readSize > 0 {
			p.buf.Write(b[:readSize])
		}
		p.ready <- err
		if err != nil {
			if err == io.EOF {
				return n, nil
			}
			return
		}
	}
}

// stdin
func (p *stdio) WriteTo(w io.Writer) (n int64, err error) {
	for {
		select {
		case <-p.ready:
			w.Write(p.readAll())
		}
	}
}

// ------------------------------------------------------------------------
// Should not be called
// ------------------------------------------------------------------------
func (p *stdio) Read(b []byte) (n int, err error) {
	n, err = p.buf.Read(b)
	if err == io.EOF {
		return n, nil
	}
	return n, err
}

func (p *stdio) Write(b []byte) (n int, err error) {
	n, err = p.buf.Write(b)
	if p.ready != nil {
		p.ready <- err
	}
	return n, err
}
