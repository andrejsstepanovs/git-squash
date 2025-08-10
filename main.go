package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/andrejsstepanovs/git-squash/cmd"
)

func main() {
	// Create context with cancel and signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cancel()
	}()

	rootCmd := cmd.NewCommand()

	err := rootCmd.ExecuteContext(ctx)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
