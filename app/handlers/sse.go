package handlers

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
)

type reloadSession struct {
	val          float64
	stateChannel chan float64
}

type sessionsLock struct {
	MU       sync.Mutex
	sessions []*reloadSession
}

func (sl *sessionsLock) addSession(s *reloadSession) {
	sl.MU.Lock()
	sl.sessions = append(sl.sessions, s)
	sl.MU.Unlock()
}

func (sl *sessionsLock) removeSession(s *reloadSession) {
	sl.MU.Lock()
	idx := slices.Index(sl.sessions, s)
	if idx != -1 {
		sl.sessions[idx] = nil
		sl.sessions = slices.Delete(sl.sessions, idx, idx+1)
	}
	sl.MU.Unlock()
}

var currentSessions sessionsLock

func formatSSEMessage(eventType string, data any) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)

	m := map[string]any{
		"data": data,
	}

	err := enc.Encode(m)
	if err != nil {
		return "", nil
	}
	sb := strings.Builder{}

	sb.WriteString(fmt.Sprintf("event: %s\n", eventType))
	sb.WriteString(fmt.Sprintf("retry: %d\n", 15000))
	sb.WriteString(fmt.Sprintf("data: %v\n\n", buf.String()))

	return sb.String(), nil
}

// nolint: funlen, staticcheck
func ReloadSSE(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	query := c.Query("query")

	log.Printf("New Request\n")

	stateChan := make(chan float64)

	val, err := strconv.ParseFloat(query, 64)
	if err != nil {
		val = 0
	}

	s := reloadSession{
		val:          val,
		stateChannel: stateChan,
	}

	currentSessions.addSession(&s)

	notify := c.Context().Done()

	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		keepAliveTickler := time.NewTicker(15 * time.Second)
		keepAliveMsg := ":keepalive\n"

		// listen to signal to close and unregister (doesn't seem to be called)
		go func() {
			<-notify
			log.Printf("Stopped Request\n")
			currentSessions.removeSession(&s)
			keepAliveTickler.Stop()
		}()

		for loop := true; loop; {
			select {

			case ev := <-stateChan:
				sseMessage, err := formatSSEMessage("current-value", ev)
				if err != nil {
					log.Printf("Error formatting sse message: %v\n", err)
					continue
				}

				// send sse formatted message
				_, err = fmt.Fprintf(w, "%s", sseMessage)

				if err != nil {
					log.Printf("Error while writing Data: %v\n", err)
					continue
				}

				err = w.Flush()
				if err != nil {
					log.Printf("Error while flushing Data: %v\n", err)
					currentSessions.removeSession(&s)
					keepAliveTickler.Stop()
					loop = false
					break
				}
			case <-keepAliveTickler.C:
				fmt.Fprintf(w, "%s", keepAliveMsg)
				err := w.Flush()
				if err != nil {
					log.Printf("Error while flushing: %v.\n", err)
					currentSessions.removeSession(&s)
					keepAliveTickler.Stop()
					loop = false
					break
				}
			}
		}

		log.Println("Exiting stream")
	}))

	return nil
}
