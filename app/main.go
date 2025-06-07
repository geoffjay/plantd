package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"sync"
	"syscall"

	cfg "github.com/geoffjay/plantd/app/config"
	"github.com/geoffjay/plantd/core"
	plog "github.com/geoffjay/plantd/core/log"

	log "github.com/sirupsen/logrus"
)

// @title Plantd Web Application
// @version 1.0
// @description The Plantd web application.
// @contact.name Geoff Johnson
// @contact.email geoff.jay@gmail.com
// @license.name MIT
// @license.url https://opensource.org/license/mit/
func main() {
	config := cfg.GetConfig()

	processArgs()
	plog.Initialize(config.Log)

	service := service{}
	service.init()

	fields := log.Fields{"service": "app", "context": "main"}

	ctx, cancelFunc := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	wg.Add(1)
	go service.run(ctx, wg)

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
			fmt.Println(core.VERSION)
		}
		os.Exit(0)
	}
}
