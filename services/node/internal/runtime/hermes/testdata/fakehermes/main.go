package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {
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
	default:
		os.Exit(8)
	}
}
