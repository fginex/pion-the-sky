package all

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/radekg/boos/configs"
	be "github.com/radekg/boos/pkg/backend"
	fe "github.com/radekg/boos/pkg/frontend"
	"github.com/spf13/cobra"
)

// Command is the command declaration.
var Command = &cobra.Command{
	Use:   "all",
	Short: "Starts a backend and a frontend server",
	Run:   run,
	Long:  ``,
}

var (
	backEndConfig  = configs.NewBackendConfig()
	frontEndConfig = configs.NewFrontendConfig()
	logConfig      = configs.NewLoggingConfig()
	webRTCConfig   = configs.NewWebRTCConfig()
)

func initFlags() {
	Command.Flags().AddFlagSet(backEndConfig.FlagSet())
	Command.Flags().AddFlagSet(frontEndConfig.FlagSet())
	Command.Flags().AddFlagSet(logConfig.FlagSet())
	Command.Flags().AddFlagSet(webRTCConfig.FlagSet())
}

func init() {
	initFlags()
}

func run(cobraCommand *cobra.Command, _ []string) {
	os.Exit(processCommand())
}

func processCommand() int {

	appLogger := logConfig.NewLogger("all")
	if err := be.ServeListen(backEndConfig, frontEndConfig, webRTCConfig, appLogger.Named("backend")); err != nil {
		appLogger.Error("Error binding backend server", "reason", err)
		return 1
	}

	if err := fe.ServeListen(backEndConfig, frontEndConfig, webRTCConfig, appLogger.Named("frontend")); err != nil {
		appLogger.Error("Error binding frontend server", "reason", err)
		return 1
	}

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT)
	<-sig

	return 0
}
