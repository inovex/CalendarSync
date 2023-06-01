package main

import (
	"fmt"
	"os"
	"time"

	"github.com/inovex/CalendarSync/internal/auth"
	"github.com/inovex/CalendarSync/internal/models"

	"github.com/charmbracelet/log"
	"github.com/urfave/cli/v2"

	"github.com/inovex/CalendarSync/internal/adapter"
	"github.com/inovex/CalendarSync/internal/config"
	"github.com/inovex/CalendarSync/internal/sync"
)

const (
	flagLogLevel             = "log-level"
	flagConfigFilePath       = "config"
	flagStorageEncryptionKey = "storage-encryption-key"
	flagClean                = "clean"
	flagDryRun               = "dry-run"
	flagPort                 = "port"
)

var (
	// The following vars are set during linking
	// Version is the version from which the binary was built.
	Version string
	// BuildTime is the timestamp of building the binary.
	BuildTime string
)

func main() {
	app := &cli.App{
		Name:        "CalendarSync",
		Usage:       "Stateless calendar sync across providers",
		Description: fmt.Sprintf("Version %s - Build date %s", Version, BuildTime),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  flagLogLevel,
				Value: log.InfoLevel.String(),
			},
			&cli.StringFlag{
				Name:  flagConfigFilePath,
				Usage: "path to the config yaml",
				Value: "sync.yaml",
			},
			&cli.StringFlag{
				Name:     flagStorageEncryptionKey,
				Usage:    "encryption string",
				Required: true,
			},
			&cli.BoolFlag{
				Name:  flagClean,
				Usage: "cleans your sink calendar from all the synced events",
				Value: false,
			},
			&cli.BoolFlag{
				Name:  flagDryRun,
				Usage: "This flag helps you see which events would get created, updated or deleted without actually doing these operations",
				Value: false,
			},
			&cli.UintFlag{
				Name:   flagPort,
				Usage:  "set manual free port for the authentication process",
				Hidden: true,
				Value:  0,
			},
		},
		Before: func(c *cli.Context) error {
			// setup global logger
			level := log.ParseLevel(c.String(flagLogLevel))
			log.SetLevel(level)
			log.SetTimeFormat(time.Kitchen)
			if level == log.DebugLevel {
				log.SetReportCaller(true)
			}
			return nil
		},
		Action: Run,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func Run(c *cli.Context) error {
	cfg, err := config.NewFromFile(c.String(flagConfigFilePath))
	if err != nil {
		return err
	}
	log.Info("loaded config file", "path", cfg.Path)

	startTime, err := models.TimeFromConfig(cfg.Sync.StartTime)
	if err != nil {
		return err
	}
	endTime, err := models.TimeFromConfig(cfg.Sync.EndTime)
	if err != nil {
		return err
	}

	log.Debug("configured start and end time for sync", "start", startTime, "end", endTime)

	var sourceBindAuthPort, sinkBindAuthPort uint
	if c.IsSet("port") {
		sourceBindAuthPort = c.Uint("port")
		sinkBindAuthPort = c.Uint("port") + 1
	}

	storage, err := auth.NewStorageAdapterFromConfig(c.Context, cfg.Auth, c.String(flagStorageEncryptionKey))
	if err != nil {
		log.Fatal("error during storage adapter load:", err)
	}

	sourceLogger := log.With("adapter", cfg.Source.Adapter.Type, "type", "source")

	sourceAdapter, err := adapter.NewSourceAdapterFromConfig(
		c.Context,
		sourceBindAuthPort,
		config.NewAdapterConfig(cfg.Source.Adapter),
		storage,
		sourceLogger,
	)
	if err != nil {
		return err
	}
	log.Info("loaded source adapter", "adapter", cfg.Source.Adapter.Type, "calendar", cfg.Source.Adapter.Calendar)

	sinkLogger := log.With("adapter", cfg.Sink.Adapter.Type, "type", "sink")

	sinkAdapter, err := adapter.NewSinkAdapterFromConfig(
		c.Context,
		sinkBindAuthPort,
		config.NewAdapterConfig(cfg.Sink.Adapter),
		storage,
		sinkLogger,
	)
	if err != nil {
		return err
	}
	log.Info("loaded sink adapter", "adapter", cfg.Sink.Adapter.Type, "calendar", cfg.Sink.Adapter.Calendar)

	if log.GetLevel() == log.DebugLevel {
		for _, transformation := range cfg.Transformations {
			log.Debug("configured transformer", "name", transformation.Name, "config", transformation.Config)
		}
	}

	controller := sync.NewController(log.Default(), sourceAdapter, sinkAdapter, sync.TransformerFactory(cfg.Transformations)...)
	if cfg.UpdateConcurrency != 0 {
		controller.SetConcurrency(cfg.UpdateConcurrency)
	}
	log.Info("loaded sync controller")

	if c.Bool("clean") {
		err = controller.CleanUp(c.Context, startTime, endTime)
		if err != nil {
			log.Fatalf("we had some errors during cleanup:\n%v", err)
		}

	} else {
		err = controller.SynchroniseTimeframe(c.Context, startTime, endTime, c.Bool("dry-run"))
		if err != nil {
			log.Fatalf("we had some errors during synchronization:\n%v", err)
		}
	}
	return nil
}
