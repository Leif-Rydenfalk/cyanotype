package cmd

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/tychonis/cyanotype/cmd/artifact"
	"github.com/tychonis/cyanotype/cmd/bom"
	"github.com/tychonis/cyanotype/cmd/build"
	"github.com/tychonis/cyanotype/cmd/commit"
	"github.com/tychonis/cyanotype/cmd/export"
	"github.com/tychonis/cyanotype/cmd/format"
	"github.com/tychonis/cyanotype/cmd/history"
	"github.com/tychonis/cyanotype/cmd/initialize"
	"github.com/tychonis/cyanotype/cmd/plan"
	"github.com/tychonis/cyanotype/cmd/pull"
	"github.com/tychonis/cyanotype/cmd/push"
	"github.com/tychonis/cyanotype/cmd/query"
	"github.com/tychonis/cyanotype/cmd/scene"
	"github.com/tychonis/cyanotype/cmd/symbol"
	"github.com/tychonis/cyanotype/cmd/tree"
	"github.com/tychonis/cyanotype/cmd/version"
)

var debug bool

var rootCmd = &cobra.Command{
	Use:   "cyanotype",
	Short: "cyanotype manages bom as code",
	Long:  "TODO: Add doc string",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if debug {
			slog.SetLogLoggerLevel(slog.LevelDebug)
			slog.Debug("Debug logging enabled")
		}
	},
}

type RunEFunc = func(*cobra.Command, []string) error

func RunWithoutUsage(run RunEFunc) RunEFunc {
	return func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return run(cmd, args)
	}
}

func silenceRunEUsage(cmd *cobra.Command) {
	if cmd.RunE != nil {
		cmd.RunE = RunWithoutUsage(cmd.RunE)
	}

	for _, child := range cmd.Commands() {
		silenceRunEUsage(child)
	}
}

func Run() {
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug logging")

	rootCmd.AddCommand(
		initialize.Cmd,
		bom.Cmd,
		build.Cmd,
		commit.Cmd,
		export.Cmd,
		tree.Cmd,
		pull.Cmd,
		push.Cmd,
		plan.Cmd,
		query.Cmd,
		scene.Cmd,
		history.Cmd,
		version.Cmd,
		symbol.Cmd,
		artifact.Cmd,
		format.Cmd,
	)
	silenceRunEUsage(rootCmd)

	err := rootCmd.Execute()
	if err != nil {
		slog.Error("Failed to execute command.", "error", err)
		os.Exit(1)
	}
	os.Exit(0)
}
