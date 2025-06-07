package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"sync"
	"syscall"

	"github.com/geoffjay/plantd/core"
	plog "github.com/geoffjay/plantd/core/log"
	"github.com/geoffjay/plantd/core/util"

	log "github.com/sirupsen/logrus"
)

func main() {
	config := GetConfig()

	processArgs()
	plog.Initialize(config.Log)

	port, _ := strconv.Atoi(util.Getenv("PLANTD_PROXY_PORT", "5000"))
	bind := util.Getenv("PLANTD_PROXY_ADDRESS", "0.0.0.0")
	app := NewService(port, bind)

	ctx, cancelFunc := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	wg.Add(1)
	go app.run(ctx, wg)

	log.Debug("service started")

	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)
	<-termChan

	log.Debug("service terminated")

	cancelFunc()
	wg.Wait()

	log.Debug("proxy exiting")
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
