package main

import (
	"log"
	"time"

	"github.com/adlane/exec"
)

func main() {
	if os, err := exec.TimeoutedExec(time.Second, "cat", "/dev/urandom"); err != nil {
		log.Fatal("command failed:", err)
	} else {
		log.Printf("%x\n", os)
	}
}
