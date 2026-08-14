package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--child" {
		for {
			time.Sleep(time.Second)
		}
	}
	if len(os.Args) != 2 || os.Args[1] != "--version" {
		os.Exit(9)
	}
	switch os.Getenv("YORVA_FAKE_HERMES_MODE") {
	case "success":
		fmt.Println("Hermes Agent v0.19.7 (2026.8.14)")
	case "failure":
		fmt.Fprintln(os.Stderr, "private fake failure detail")
		os.Exit(3)
	case "output-limit":
		fmt.Print(strings.Repeat("x", 70*1024))
	case "wait":
		if path := os.Getenv("YORVA_FAKE_HERMES_PID_FILE"); path != "" {
			_ = os.WriteFile(path, []byte(fmt.Sprintf("%d", os.Getpid())), 0o600)
		}
		for {
			time.Sleep(time.Second)
		}
	case "child-wait":
		child := exec.Command(os.Args[0], "--child")
		if err := child.Start(); err != nil {
			os.Exit(7)
		}
		if path := os.Getenv("YORVA_FAKE_HERMES_PID_FILE"); path != "" {
			_ = os.WriteFile(path, []byte(fmt.Sprintf("%d", child.Process.Pid)), 0o600)
		}
		for {
			time.Sleep(time.Second)
		}
	case "child-exit":
		child := exec.Command(os.Args[0], "--child")
		if err := child.Start(); err != nil {
			os.Exit(7)
		}
		if path := os.Getenv("YORVA_FAKE_HERMES_PID_FILE"); path != "" {
			_ = os.WriteFile(path, []byte(fmt.Sprintf("%d", child.Process.Pid)), 0o600)
		}
	default:
		os.Exit(8)
	}
}
