// Package main provides the main entry point for the PlantD state service.
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

	ctx, cancelFunc := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	wg.Add(1)
	go service.Run(ctx, wg)

	log.WithFields(log.Fields{"context": "main"}).Debug("starting")

	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)
	<-termChan

	log.WithFields(log.Fields{"context": "main"}).Debug("terminated")

	cancelFunc()
	wg.Wait()

	log.WithFields(log.Fields{"context": "main"}).Debug("exiting")
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
