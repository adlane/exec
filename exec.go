package exec

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"time"
)

// TimeoutedExec executes a timeouted command.
// The program path is defined by the name arguments, args are passed as arguments to the program.
//
// TimeoutedExec returns process output as a string (stdout) , and stderr as an error.
func TimeoutedExec(timeout time.Duration, name string, args ...string) (string, error) {
	c := exec.Command(name, args...)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	c.Stdout = stdout
	c.Stderr = stderr

	if err := c.Start(); err != nil {
		return "", err
	}

	done := make(chan error, 1)
	go func() {
		_, err := c.Process.Wait()
		done <- err
	}()
	select {
	case <-time.After(timeout):
		c.Process.Signal(os.Kill)
	case <-done:
	}

	res := string(stdout.Bytes())
	if err := string(stderr.Bytes()); len(err) > 0 {
		return res, errors.New(err)
	}
	return res, nil
}
