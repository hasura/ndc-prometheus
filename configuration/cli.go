package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/hasura/ndc-prometheus/configuration/version"
	"github.com/lmittmann/tint"
)

// UpdateArguments represent input arguments of the `update` command
type UpdateArguments struct {
	Dir string `help:"The directory where the configuration.yaml file is present" short:"d" env:"HASURA_PLUGIN_CONNECTOR_CONTEXT_PATH" default:"."`
}

var cli struct {
	LogLevel string          `help:"Log level." enum:"debug,info,warn,error,DEBUG,INFO,WARN,ERROR" env:"HASURA_PLUGIN_LOG_LEVEL" default:"info"`
	Update   UpdateArguments `cmd:"" help:"Introspect metric metadata and update configuration."`
	Version  struct{}        `cmd:"" help:"Print the CLI version."`
}

func main() {
	// Handle SIGINT (CTRL+C) gracefully.
	ctx, stop := signal.NotifyContext(context.TODO(), os.Interrupt)
	defer stop()

	cmd := kong.Parse(&cli, kong.UsageOnError())
	_, err := initLogger(cli.LogLevel)
	if err != nil {
		log.Fatalf("failed to initialize: %s", err)
	}
	switch cmd.Command() {
	case "update":
		if err := introspectSchema(ctx, &cli.Update); err != nil {
			log.Fatalf("failed to update configuration: %s", err)
		}
	case "version":
		_, _ = fmt.Print(version.BuildVersion)
	default:
		log.Fatalf("unknown command <%s>", cmd.Command())
	}
}

func initLogger(logLevel string) (*slog.Logger, error) {
	var level slog.Level
	err := level.UnmarshalText([]byte(strings.ToUpper(logLevel)))
	if err != nil {
		return nil, err
	}

	logger := slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level:      level,
		TimeFormat: "15:04",
	}))
	slog.SetDefault(logger)

	return logger, nil
}
