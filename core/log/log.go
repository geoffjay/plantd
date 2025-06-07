// Package log provides logging initialization functionality.
package log

import (
	"github.com/geoffjay/plantd/core/config"

	log "github.com/sirupsen/logrus"
	loki "github.com/yukitsune/lokirus"
)

// Initialize sets up the logging configuration.
func Initialize(logConfig config.LogConfig) {
	if logLevel, err := log.ParseLevel(logConfig.Level); err == nil {
		log.SetLevel(logLevel)
	}

	if logConfig.Formatter == "json" {
		log.SetFormatter(&log.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
		})
	} else {
		log.SetFormatter(&log.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
	}

	opts := loki.NewLokiHookOptions().WithLevelMap(
		loki.LevelMap{log.PanicLevel: "critical"},
	).WithFormatter(
		&log.JSONFormatter{},
	).WithStaticLabels(
		logConfig.Loki.Labels,
	)

	hook := loki.NewLokiHookWithOpts(
		logConfig.Loki.Address,
		opts,
		// log.DebugLevel,
		log.InfoLevel,
		log.WarnLevel,
		log.ErrorLevel,
		log.FatalLevel,
	)

	log.AddHook(hook)
}
