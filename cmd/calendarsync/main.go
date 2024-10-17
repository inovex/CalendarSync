package main

import (
	"fmt"
	"os"
	"runtime"
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
	flagLogLevel                 = "log-level"
	flagConfigFilePath           = "config"
	flagStorageEncryptionKey     = "storage-encryption-key"
	flagClean                    = "clean"
	flagDryRun                   = "dry-run"
	flagPort                     = "port"
	flagOpenBrowserAutomatically = "open-browser"
	flagVersion                  = "version"
)

var (
	// The following vars are set during linking
	// Version is the version from which the binary was built.
	Version string
)

func main() {
	app := &cli.App{
		Name:        "CalendarSync",
		Usage:       "Stateless calendar sync across providers",
		Description: fmt.Sprintf("Version: %s", Version),
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
				Name:  flagStorageEncryptionKey,
				Usage: "encryption string to be used for encrypting the local auth-storage file. NOTE: This option is deprecated. Please use the CALENDARSYNC_ENCRYPTION_KEY env variable. The flag will be removed in later versions",
			},
			&cli.BoolFlag{
				Name:  flagOpenBrowserAutomatically,
				Usage: "opens the browser automatically for the authentication process",
				Value: true,
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
			&cli.BoolFlag{
				Name:  flagVersion,
				Usage: "shows the version of CalendarSync",
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
			level, err := log.ParseLevel(c.String(flagLogLevel))
			if err != nil {
				return err
			}
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
	if c.Bool(flagVersion) {
		fmt.Println("Version:", Version)
		os.Exit(0)
	}

	cfg, err := config.NewFromFile(c.String(flagConfigFilePath))
	if err != nil {
		return err
	}
	log.Info("loaded config file", "path", cfg.Path)

	if len(c.String(flagStorageEncryptionKey)) > 0 {
		log.Warn("Parsing the encryption key using the flag is deprecated. Please use the environment variable $CALENDARSYNC_ENCRYPTION_KEY instead.")
	} else {
		if encKeyEnv, envSet := os.LookupEnv("CALENDARSYNC_ENCRYPTION_KEY"); envSet {
			err := c.Set(flagStorageEncryptionKey, encKeyEnv)
			if err != nil {
				return err
			}
		}
	}

	if len(c.String(flagStorageEncryptionKey)) == 0 {
		return fmt.Errorf("storage encryption key needs to be set")
	}

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
		log.Fatal("error during storage adapter load", "error", err)
	}

	sourceLogger := log.With("adapter", cfg.Source.Adapter.Type, "type", "source")

	sourceAdapter, err := adapter.NewSourceAdapterFromConfig(
		c.Context,
		sourceBindAuthPort,
		c.Bool(flagOpenBrowserAutomatically),
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
		c.Bool(flagOpenBrowserAutomatically),
		config.NewAdapterConfig(cfg.Sink.Adapter),
		storage,
		sinkLogger,
	)
	if err != nil {
		return err
	}
	log.Info("loaded sink adapter", "adapter", cfg.Sink.Adapter.Type, "calendar", cfg.Sink.Adapter.Calendar)

	// By default go runs a garbage collection once the memory usage doubles compared to the last GC run.
	// Decrypting the storage in NewSourceAdapterFromConfig/NewSinkAdapterFromConfig requires a lot of memory,
	// such that the next GC only trigger once the memory usage double compared to that peak. Explicitly trigger
	// a GC to reset the memory usage reference level.
	runtime.GC()

	if log.GetLevel() == log.DebugLevel {
		for _, transformation := range cfg.Transformations {
			log.Debug("configured transformer", "name", transformation.Name, "config", transformation.Config)
		}
	}

	controller := sync.NewController(log.Default(), sourceAdapter, sinkAdapter, sync.TransformerFactory(cfg.Transformations), sync.FilterFactory(cfg.Filters))
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
