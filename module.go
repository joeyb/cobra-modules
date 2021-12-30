package cobramodules

import (
	"context"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/multierr"
)

// Module defines the interface used to segregate a command's subsystems. Each module is started in its own goroutine
// and is expected to run until either an unrecoverable error occurs or the given context.Context is cancelled.
type Module interface {

	// BindFlags gives modules the opportunity to add module-specific command-line flags. Due to the way that binding
	// works with pointers, it's important to implement the Module interface with pointer receivers to ensure that pflag
	// binds to the correct instances. If you see that argument values are not being set correctly, that's a good sign
	// that the module instance is being passed by value somewhere.
	BindFlags(flagSet *pflag.FlagSet)

	// Start launches the module and is expected to block until either an unrecoverable error occurs or the given
	// context.Context is cancelled. Cleaning up and returning after the context.Context is cancelled is important
	// because it is expected that the module runner will wait for all modules to return before it can return.
	Start(ctx context.Context) error
}

// NewModuleRunner returns a func that matches the signature of the cobra.Command RunE function. When executed, that
// returned func will start all the given modules.
func NewModuleRunner(ctx context.Context, modules []Module) func(c *cobra.Command, args []string) error {
	return func(c *cobra.Command, args []string) error {
		return RunModules(ctx, modules)
	}
}

// RunModules starts all the given modules in their own goroutine. Each module is expected to run until either an
// unrecoverable error occurs or the given context.Context is cancelled. Any errors returned by the modules are
// aggregated and returned as a together as a multierr.
func RunModules(ctx context.Context, modules []Module) error {
	// We use a channel to feed error responses from the modules back up to this command. The channel is closed
	// once all the modules have stopped.
	errCh := make(chan error)

	// If one of the modules stops (with or without an error), we need to tell the other modules to finish what
	// they are doing and stop as well. The context is closed whenever the first module finishes.
	ctx, cancel := context.WithCancel(ctx)
	stopCloseOnce := sync.Once{}

	// The outer goroutine handles launching goroutines for each module, then waits for them all to complete.
	// Each module's goroutine starts the module, publishes any errors on completion, then closes the stop
	// channel to instruct the rest of the modules to shut down.
	go func() {
		wg := sync.WaitGroup{}
		wg.Add(len(modules))

		for _, m := range modules {
			go func(m Module) {
				err := m.Start(ctx)

				if err != nil {
					errCh <- err
				}

				wg.Done()
				stopCloseOnce.Do(func() { cancel() })
			}(m)
		}

		wg.Wait()
		close(errCh)
	}()

	var result error

	for err := range errCh {
		result = multierr.Append(result, err)
	}

	return result
}

// BindModuleFlags iterates over the given modules and calls Module.BindFlags.
func BindModuleFlags(modules []Module, c *cobra.Command) {
	for _, m := range modules {
		m.BindFlags(c.Flags())
	}
}
