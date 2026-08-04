package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"

	"github.com/robinjanke/tart-orchard-gitlab-runner/internal/commands"
	"github.com/robinjanke/tart-orchard-gitlab-runner/internal/gitlab"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)
	go func() {
		select {
		case <-interruptCh:
			cancel()
		case <-ctx.Done():
		}
	}()

	// GitLab Runner already timestamps custom executor logs.
	log.SetFlags(0)

	buildFailureExitCode := gitlab.ParseExitCode("BUILD_FAILURE_EXIT_CODE", 1)
	systemFailureExitCode := gitlab.ParseExitCode("SYSTEM_FAILURE_EXIT_CODE", 2)

	if err := commands.NewRootCmd().ExecuteContext(ctx); err != nil {
		log.Println(err)

		var systemFailureError *gitlab.SystemFailureError
		if errors.As(err, &systemFailureError) {
			os.Exit(systemFailureExitCode)
		}
		os.Exit(buildFailureExitCode)
	}
}
