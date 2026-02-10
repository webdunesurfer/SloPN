//go:build windows

package main

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/sys/windows/svc"
)

type winsvc struct {
	h *Helper
}

func (m *winsvc) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the helper in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- m.h.run(ctx)
	}()

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

loop:
	for {
		select {
		case err := <-errChan:
			logHelper(fmt.Sprintf("Helper core stopped: %v", err))
			break loop
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				logHelper("Service stop/shutdown requested")
				cancel()
				break loop
			default:
				logHelper(fmt.Sprintf("Unexpected control request #%d", c))
			}
		}
	}

	changes <- svc.Status{State: svc.StopPending}
	return
}

func main() {
	h := &Helper{state: "disconnected"}

	// Check if we are running as a service
	isInt, err := svc.IsAnInteractiveSession()
	if err != nil {
		logHelper(fmt.Sprintf("Failed to determine if session is interactive: %v", err))
		os.Exit(1)
	}

	if !isInt {
		// Run as Windows Service
		err = svc.Run("SloPNHelper", &winsvc{h: h})
		if err != nil {
			logHelper(fmt.Sprintf("Service failed: %v", err))
			os.Exit(1)
		}
		return
	}

	// Run interactively (for development/testing)
	logHelper(fmt.Sprintf("SloPN Helper Starting Interactively. PID: %d", os.Getpid()))
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Simple way to handle Ctrl+C in interactive mode on Windows
	go func() {
		fmt.Println("Press ENTER to stop...")
		var b [1]byte
		os.Stdin.Read(b[:])
		cancel()
	}()

	if err := h.run(ctx); err != nil {
		logHelper(fmt.Sprintf("CRITICAL: %v", err))
		os.Exit(1)
	}
	
	logHelper("Helper exited gracefully.")
}
