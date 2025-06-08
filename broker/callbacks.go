// Package main provides callback handlers for the PlantD broker service.
package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/geoffjay/plantd/core/service"

	log "github.com/sirupsen/logrus"
)

type serviceCallback struct {
	name string
}

type servicesCallback struct {
	name string
}

// nolint: unused
type sinkCallback struct{}

// Execute callback function to handle `service` requests.
func (cb *serviceCallback) Execute(msgBody string) ([]byte, error) {
	var (
		scope   string
		found   bool
		request service.RawRequest
	)

	log.Tracef("name: %s", cb.name)
	log.Tracef("body: %s", msgBody)

	if err := json.Unmarshal([]byte(msgBody), &request); err != nil {
		msg := fmt.Sprintf("{\"error\":\"%s\"}", err.Error())
		return []byte(msg), err
	}

	if scope, found = request["service"].(string); !found {
		return []byte(`{"error": "service required for service request"}`),
			errors.New("`service` missing")
	}

	// TODO:
	// 1. Check if service exists
	// 2. Check if service is running
	// 3. Create response with service data, eg. client and worker count

	return []byte(`{"` + scope + `": {} }`), nil
}

// Execute callback function to handle `services` requests.
func (cb *servicesCallback) Execute(msgBody string) ([]byte, error) {
	var (
		request service.RawRequest
	)

	log.Tracef("name: %s", cb.name)
	log.Tracef("body: %s", msgBody)

	if err := json.Unmarshal([]byte(msgBody), &request); err != nil {
		msg := fmt.Sprintf("{\"error\":\"%s\"}", err.Error())
		return []byte(msg), err
	}

	// TODO:
	// 1. Check if service exists
	// 2. Check if service is running
	// 3. Create response with services data, eg. client and worker count

	return []byte(`{"services": [] }`), nil
}

// Callback handles subscriber events on the state bus.
// nolint: unused
func (cb *sinkCallback) Handle(data []byte) error {
	log.WithFields(log.Fields{"data": string(data)}).Debug(
		"data received on state bus")
	return nil
}
