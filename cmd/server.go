package cmd

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/wtsi-hgi/softpack/backend"
	"github.com/wtsi-hgi/softpack/config"
)

var RootCmd = &cobra.Command{
	Use: "softpack",
}
var configPath string

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

func init() {
	RootCmd.AddCommand(serverCmd)

	serverCmd.Flags().StringVarP(&configPath, "config", "c", "", "config")

	serverCmd.MarkFlagRequired("config") //nolint:errcheck
}

var serverCmd = &cobra.Command{
	Use: "server",
	RunE: func(_ *cobra.Command, _ []string) error {
		conf, err := config.Load(configPath)
		if err != nil {
			return err
		}

		return backend.New(conf).Run()
	},
}
