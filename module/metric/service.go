package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/geoffjay/plantd/core/bus"
	"github.com/geoffjay/plantd/core/service"
	"github.com/geoffjay/plantd/core/util"

	log "github.com/sirupsen/logrus"
)

// Service type for REST API.
type Service interface {
	Run(ctx context.Context, wg *sync.WaitGroup)
}

type Consumer struct {
	// Service
	clientEndpoint string
	client         *service.Client
	sink           *bus.Sink
}

type Producer struct {
	// Service
	clientEndpoint string
	client         *service.Client
	source         *bus.Source
}

type WeatherResponse struct {
	Latitude  float64                `json:"latitude"`
	Longitude float64                `json:"longitude"`
	Elevation float64                `json:"elevation"`
	Units     map[string]string      `json:"current_units"`
	Values    map[string]interface{} `json:"current"`
}

// NewService constructs and instance of a service type.
func NewService(serviceType string) Service {
	clientEndpoint := util.Getenv("PLANTD_MODULE_ECHO_BROKER_ENDPOINT", "tcp://127.0.0.1:9797")

	if serviceType == "consumer" {
		return NewConsumer(clientEndpoint)
	} else if serviceType == "producer" {
		return NewProducer(clientEndpoint)
	}

	log.Panic("invalid service type provided")
	return nil
}

func NewConsumer(clientEndpoint string) Service {
	return &Consumer{
		clientEndpoint: clientEndpoint,
	}
}

func NewProducer(clientEndpoint string) Service {
	return &Producer{
		clientEndpoint: clientEndpoint,
	}
}

func (c *Consumer) Run(ctx context.Context, wg *sync.WaitGroup) {
	var err error

	c.sink = bus.NewSink(">tcp://localhost:13001", "org.plantd")
	c.client, err = service.NewClient(c.clientEndpoint)
	if err != nil {
		log.Error(err)
	}

	defer c.sink.Stop()
	defer c.client.Close()

	serviceDiscoveryInform(c.client, "consumer")
	c.sink.SetHandler(&bus.SinkHandler{Callback: &metricSinkCallback{}})

	go c.sink.Run(ctx, wg)

	<-ctx.Done()

	log.WithFields(log.Fields{"module": "metric", "context": "consumer"}).Debug("exiting")
}

func (p *Producer) Run(ctx context.Context, wg *sync.WaitGroup) {
	var err error

	p.source = bus.NewSource(">tcp://localhost:13000", "org.plantd.Metric")
	p.client, err = service.NewClient(p.clientEndpoint)
	if err != nil {
		log.Error(err)
	}

	defer p.source.Shutdown()
	defer p.client.Close()

	serviceDiscoveryInform(p.client, "producer")

	go p.source.Run(ctx, wg)

	go func() {
		for range time.Tick(5 * time.Second) {
			if !p.source.Running() {
				break
			}

			baseURL := "https://api.open-meteo.com"
			resource := "/v1/forecast"
			params := url.Values{}
			weatherFields := []string{
				"temperature_2m",
				"relative_humidity_2m",
				"pressure_msl",
				"wind_speed_10m",
				"wind_direction_10m",
			}
			params.Add("latitude", "49.4614")
			params.Add("longitude", "-123.7186")
			params.Add("current", strings.Join(weatherFields, ","))

			u, _ := url.ParseRequestURI(baseURL)
			u.Path = resource
			u.RawQuery = params.Encode()
			urlStr := fmt.Sprintf("%v", u)

			resp, err := http.Get(urlStr)
			if err != nil {
				log.Error(err)
				continue
			}

			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Fatal(err)
			}

			var weatherResponse WeatherResponse
			err = json.Unmarshal(bodyBytes, &weatherResponse)
			if err != nil {
				log.Error(err)
				continue
			}

			log.Debugf("temperature: %f %s", weatherResponse.Values["temperature_2m"], weatherResponse.Units["temperature_2m"])
			log.Debugf("humidity: %f %s", weatherResponse.Values["relative_humidity_2m"], weatherResponse.Units["relative_humidity_2m"])
			log.Debugf("pressure: %f %s", weatherResponse.Values["pressure_msl"], weatherResponse.Units["pressure_msl"])
			log.Debugf("wind speed: %f %s", weatherResponse.Values["wind_speed_10m"], weatherResponse.Units["wind_speed_10m"])
			log.Debugf("wind direction: %f %s", weatherResponse.Values["wind_direction_10m"], weatherResponse.Units["wind_direction_10m"])

			metric := Metric{
				Timestamp: time.Now().String(),
				Value:     "1",
				Units:     "count",
				Tags:      []string{"weather"},
			}
			message, err := json.Marshal(metric)
			if err != nil {
				log.Error(err)
				continue
			}
			log.Trace(string(message))
			p.source.QueueMessage([]byte(message))
		}
	}()

	<-ctx.Done()

	log.WithFields(log.Fields{"module": "metric", "context": "producer"}).Debug("exiting")
}

func serviceDiscoveryInform(client *service.Client, serviceType string) {
	request := &service.RawRequest{
		"service": "org.plantd.module.Metric",
		"type":    serviceType,
	}
	_, err := client.SendRawRequest("org.plantd.ServiceRegistry", "inform", request)
	if err != nil {
		log.Error(err)
	}
}
