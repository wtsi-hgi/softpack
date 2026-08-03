package cmd

import (
	"github.com/spf13/cobra"
	"github.com/wtsi-hgi/softpack/backend"
	"github.com/wtsi-hgi/softpack/config"
)

var RootCmd = &cobra.Command{
	Use: "softpack",
}
var configPath string

func init() {
	RootCmd.AddCommand(serverCmd)

	serverCmd.Flags().StringVarP(&configPath, "config", "c", "", "config")

	serverCmd.MarkFlagRequired("config")
}

var serverCmd = &cobra.Command{
	Use: "server",
	RunE: func(cmd *cobra.Command, args []string) error {
		conf, err := config.Load(configPath)
		if err != nil {
			return err
		}

		return backend.New(conf).Run()
	},
}
