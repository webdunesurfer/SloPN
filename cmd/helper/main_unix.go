//go:build !windows

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	logHelper(fmt.Sprintf("SloPN Helper Starting. PID: %d", os.Getpid()))
	
	h := &Helper{state: "disconnected"}
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		<-sigChan
		logHelper("Shutdown signal received.")
		cancel()
	}()

	if err := h.run(ctx); err != nil {
		logHelper(fmt.Sprintf("CRITICAL: %v", err))
		os.Exit(1)
	}
	
	logHelper("Helper exited gracefully.")
}
