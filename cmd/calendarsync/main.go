package main

import (
	"fmt"
	"os"

	"gitlab.inovex.de/inovex-calendarsync/calendarsync/internal/auth"
	"gitlab.inovex.de/inovex-calendarsync/calendarsync/internal/models"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	"gitlab.inovex.de/inovex-calendarsync/calendarsync/internal/adapter"
	"gitlab.inovex.de/inovex-calendarsync/calendarsync/internal/config"
	"gitlab.inovex.de/inovex-calendarsync/calendarsync/internal/sync"
)

const (
	flagLogLevel             = "log-level"
	flagConfigFilePath       = "config"
	flagStorageEncryptionKey = "storage-encryption-key"
	flagClean                = "clean"
	flagDryRun               = "dry-run"
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
		},
		Before: func(c *cli.Context) error {
			// setup global logger
			level, err := log.ParseLevel(c.String(flagLogLevel))
			if err != nil {
				return err
			}
			log.SetLevel(level)

			return nil
		},
		Action: Run,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalln(err)
	}
}

func Run(c *cli.Context) error {
	cfg, err := config.NewFromFile(c.String(flagConfigFilePath))
	if err != nil {
		return err
	}
	log.WithField("path", cfg.Path).Infoln("loaded config file")

	startTime, err := models.TimeFromConfig(cfg.Sync.StartTime)
	if err != nil {
		return err
	}
	endTime, err := models.TimeFromConfig(cfg.Sync.EndTime)
	if err != nil {
		return err
	}

	log.WithField("start", startTime).Debug("configured start time for sync")
	log.WithField("end", endTime).Debug("configured end time for sync")

	storage, err := auth.NewStorageAdapterFromConfig(c.Context, cfg.Auth, c.String(flagStorageEncryptionKey))
	if err != nil {
		log.Fatalln("error during storage adapter load:", err)
	}

	sourceAdapter, err := adapter.NewSourceAdapterFromConfig(
		c.Context,
		config.NewAdapterConfig(cfg.Source.Adapter),
		storage,
	)
	if err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"adapter":  cfg.Source.Adapter.Type,
		"calendar": cfg.Source.Adapter.Calendar,
	}).Info("loaded source adapter")

	sinkAdapter, err := adapter.NewSinkAdapterFromConfig(
		c.Context,
		config.NewAdapterConfig(cfg.Sink.Adapter),
		storage,
	)
	if err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"adapter":  cfg.Sink.Adapter.Type,
		"calendar": cfg.Sink.Adapter.Calendar,
	}).Info("loaded sink adapter")

	if log.GetLevel() == log.DebugLevel {
		for _, transformation := range cfg.Transformations {
			log.WithFields(log.Fields{
				"name":   transformation.Name,
				"config": transformation.Config,
			}).Debug("configured transformer")
		}
	}

	controller := sync.NewController(sourceAdapter, sinkAdapter, sync.TransformerFactory(cfg.Transformations)...)
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
