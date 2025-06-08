package main

import (
	"encoding/json"
	"regexp"

	log "github.com/sirupsen/logrus"
)

type metricSinkCallback struct{}

type Metric struct {
	Timestamp string   `json:"timestamp"`
	Value     string   `json:"value"`
	Units     string   `json:"units"`
	Tags      []string `json:"tags"`
}

type MetricGroup struct {
	Metrics []Metric `json:"metrics"`
}

func (cb *metricSinkCallback) Handle(data []byte) error {
	var metric Metric

	re := regexp.MustCompile(".*{")
	message := re.ReplaceAllString(string(data), "{")
	if err := json.Unmarshal([]byte(message), &metric); err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"timestamp": metric.Timestamp,
		"value":     metric.Value,
	}).Debug("handler received metric")

	return nil
}
