// Package main provides the main entry point for the PlantD broker service.
package main

import (
	"context"
	"os"
	"os/signal"
	"regexp"
	"sync"
	"syscall"

	"github.com/geoffjay/plantd/core"
	plog "github.com/geoffjay/plantd/core/log"

	log "github.com/sirupsen/logrus"
)

func main() {
	config := GetConfig()

	processArgs()
	plog.Initialize(config.Log)

	service := NewService()
	fields := log.Fields{"service": "broker", "context": "main"}

	ctx, cancelFunc := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	wg.Add(1)
	go service.Run(ctx, wg)

	log.WithFields(fields).Debug("starting")

	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)
	<-termChan

	log.WithFields(fields).Debug("terminated")

	cancelFunc()
	wg.Wait()

	log.WithFields(fields).Debug("exiting")
}

func processArgs() {
	if len(os.Args) > 1 {
		r := regexp.MustCompile("^-V$|(-{2})?version$")
		if r.Match([]byte(os.Args[1])) {
			log.Info(core.VERSION)
		}
		os.Exit(0)
	}
}
