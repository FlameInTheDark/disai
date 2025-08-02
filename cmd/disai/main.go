package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v3"

	"github.com/FlameInTheDark/disai/internal/config"
)

func main() {
	cmd := &cli.Command{
		Name:        "disai",
		Description: "Ollama MCP tool bot",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"cfg"},
				Usage:   "config file path",
				Value:   "./config.yaml",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			cfg := config.NewConfig(c.String("config"))
			app := NewApp(cfg)
			err := app.Run()
			if err != nil {
				return err
			}
			app.createCommands()
			app.registerHandlers()
			slog.Info("Up and running")
			signalCh := make(chan os.Signal, 1)
			signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
			<-signalCh
			return nil
		},
	}
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		panic(err)
	}
}
