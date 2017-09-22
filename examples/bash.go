package main

import (
	"fmt"
	"time"

	"github.com/adlane/exec"
)

func main() {
	ctx := exec.InteractiveExec("bash", "-i")
	r := reader{}
	go ctx.Receive(&r, 5*time.Second)
	ctx.Send("echo hello world\n")
	time.Sleep(time.Second)
	ctx.Send("ls\n")
	time.Sleep(time.Second)
}

type reader struct {
}

func (*reader) OnData(b []byte) bool {
	fmt.Print(string(b))
	return false
}

func (*reader) OnError(b []byte) bool {
	fmt.Print(string(b))
	return false
}

func (*reader) OnTimeout() {}
