package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	log "github.com/sirupsen/logrus"
)

type reloadClient struct {
	events chan string
}

func Reload(w http.ResponseWriter, _ *http.Request) {
	client := &reloadClient{events: make(chan string, 10)}
	go publishReload(client)

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Cache-Control")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")

	for event := range client.events {
		// select {
		// case event := <-client.events:
		// event := <-client.events
		fmt.Fprintf(w, "data: %s\n\n", event)
		log.Debugf("data: %s\n", event)
		// }

		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
}

func publishReload(client *reloadClient) {
	for {
		client.events <- time.Now().Format(time.TimeOnly)
		time.Sleep(10 * time.Second)
	}
}

func UpgradeWS(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		c.Locals("allowed", true)
		return c.Next()
	}

	return fiber.ErrUpgradeRequired
}

func ReloadWS(c *websocket.Conn) {
	for {
		// mt, msg, err := c.ReadMessage()
		// if err != nil {
		// 	log.Println("read:", err)
		// 	break
		// }
		// log.Printf("recv: %s", msg)
		timestamp := time.Now().Format(time.RFC3339)
		msg := fmt.Sprintf("<div id=message>%s</div>", timestamp)
		err := c.WriteMessage(0, []byte(msg))
		if err != nil {
			log.Println("write:", err)
			break
		}

		time.Sleep(10 * time.Second)
	}
}
