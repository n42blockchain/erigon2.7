//go:build windows

package debug

import (
	"io"
	"os"
	"os/signal"

	"github.com/erigontech/erigon-lib/log/v3"
	_debug "github.com/erigontech/erigon/common/debug"
)

func ListenSignals(stack io.Closer, logger log.Logger) {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt)
	_debug.GetSigC(&sigc)
	defer signal.Stop(sigc)

	<-sigc
	logger.Info("Got interrupt, shutting down...")
	if stack != nil {
		// Close synchronously to ensure all data is flushed before exit
		closeDone := make(chan struct{})
		go func() {
			stack.Close()
			close(closeDone)
		}()
		// Wait for close to complete or force exit on repeated interrupts
		forceExitCount := 3
		for {
			select {
			case <-closeDone:
				logger.Info("Graceful shutdown completed")
				Exit()
				return
			case <-sigc:
				forceExitCount--
				if forceExitCount <= 0 {
					logger.Warn("Force exiting...")
					Exit()
					LoudPanic("forced exit")
				}
				logger.Warn("Still shutting down, interrupt more to force exit", "times", forceExitCount)
			}
		}
	}
	Exit() // ensure trace and CPU profile data is flushed.
}
