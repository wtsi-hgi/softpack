package cmd

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

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

		b, err := backend.New(conf)
		if err != nil {
			return err
		}

		go b.Run()

		return handleShutdown(b)
	},
}

func handleShutdown(b *backend.Server) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()

	slog.Info("Received termination signal, shutting down...")

	if err := b.Close(); err != nil {
		slog.Error("Failure closing server", "err", err)

		return err
	}

	return nil
}
