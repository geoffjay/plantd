package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/geoffjay/plantd/core/util"

	log "github.com/sirupsen/logrus"
	loki "github.com/yukitsune/lokirus"
)

func main() {
	initLogging()

	serviceType := util.Getenv("PLANTD_MODULE_METRIC_TYPE", "consumer")
	fields := log.Fields{"module": fmt.Sprintf("metric-%s", serviceType)}
	service := NewService(serviceType)

	ctx, cancelFunc := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	wg.Add(1)
	go service.Run(ctx, wg)

	log.WithFields(fields).Debug("started")

	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)
	<-termChan

	log.WithFields(fields).Debug("terminated")

	cancelFunc()
	wg.Wait()

	log.WithFields(fields).Debug("exiting")
}

func initLogging() {
	level := util.Getenv("PLANTD_MODULE_METRIC_LOG_LEVEL", "info")
	if logLevel, err := log.ParseLevel(level); err == nil {
		log.SetLevel(logLevel)
	}

	format := util.Getenv("PLANTD_MODULE_METRIC_LOG_FORMAT", "text")
	if format == "json" {
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
		loki.Labels{
			"app":         "module-metric",
			"environment": "development",
		},
	)

	hook := loki.NewLokiHookWithOpts(
		"http://localhost:3100",
		opts,
		log.InfoLevel,
		log.WarnLevel,
		log.ErrorLevel,
		log.FatalLevel,
	)

	log.AddHook(hook)
}
