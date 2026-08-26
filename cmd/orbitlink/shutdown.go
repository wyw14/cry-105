package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func shutdownContext() (context.Context, func()) {
	root, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	return root, stop
}

func closeContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}
