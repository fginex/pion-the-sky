package backend

import (
	"os/signal"
	"syscall"

	"github.com/radekg/boos/configs"
	be "github.com/radekg/boos/pkg/backend"
	"github.com/spf13/cobra"

	"os"
)

// Command is the command declaration.
var Command = &cobra.Command{
	Use:   "backend",
	Short: "Starts a backend server",
	Run:   run,
	Long:  ``,
}

var (
	backEndConfig  = configs.NewBackEndConfig()
	frontEndConfig = configs.NewFrontEndConfig()
	logConfig      = configs.NewLoggingConfig()
)

func initFlags() {
	Command.Flags().AddFlagSet(backEndConfig.FlagSet())
	Command.Flags().AddFlagSet(frontEndConfig.FlagSet())
	Command.Flags().AddFlagSet(logConfig.FlagSet())
}

func init() {
	initFlags()
}

func run(cobraCommand *cobra.Command, _ []string) {
	os.Exit(processCommand())
}

func processCommand() int {
	logger := logConfig.NewLogger("backend")
	if err := be.ServeListen(backEndConfig, frontEndConfig, logger); err != nil {
		logger.Error("Error binding backend server", "reason", err)
		return 1
	}

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT)
	<-sig

	return 0
}
