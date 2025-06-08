// Package main provides callback handlers for the PlantD state service.
package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/geoffjay/plantd/core/service"

	log "github.com/sirupsen/logrus"
)

type createScopeCallback struct {
	name    string
	store   *Store
	manager *Manager
}

type deleteScopeCallback struct {
	name    string
	store   *Store
	manager *Manager
}

type deleteCallback struct {
	name  string
	store *Store
}

type getCallback struct {
	name  string
	store *Store
}

type setCallback struct {
	name  string
	store *Store
}

type sinkCallback struct {
	store *Store
}

// Execute callback function to handle `create-scope` requests.
func (cb *createScopeCallback) Execute(msgBody string) ([]byte, error) {
	var (
		scope   string
		found   bool
		request service.RawRequest
	)

	log.Tracef("name: %s", cb.name)
	log.Tracef("body: %s", msgBody)

	if err := json.Unmarshal([]byte(msgBody), &request); err != nil {
		return []byte(`{"error": "` + err.Error() + `"}`), err
	}

	if scope, found = request["service"].(string); !found {
		err := errors.New("`service` missing")
		return []byte(`{"error": "service required for create-scope request"}`),
			err
	}

	if cb.store.HasScope(scope) {
		// this shouldn't fail, just report to the caller
		msg := fmt.Sprintf("{\"error\":\"the scope %s already exists\"}", scope)
		return []byte(msg), nil
	}

	err := cb.store.CreateScope(scope)
	if err != nil {
		msg := fmt.Sprintf("{\"error\":\"%s\"}", err.Error())
		return []byte(msg), err
	}

	// if the bucket for the scope was successfully created add a sink to listen
	// for events
	cb.manager.AddSink(scope, &sinkCallback{store: cb.store})

	return []byte("{}"), nil
}

// Execute callback function to handle `delete-scope` requests.
func (cb *deleteScopeCallback) Execute(msgBody string) ([]byte, error) {
	var (
		scope   string
		found   bool
		request service.RawRequest
	)

	log.Tracef("name: %s", cb.name)
	log.Tracef("body: %s", msgBody)

	if err := json.Unmarshal([]byte(msgBody), &request); err != nil {
		return []byte(`{"error": "` + err.Error() + `"}`), err
	}

	if scope, found = request["service"].(string); !found {
		err := errors.New("`service` missing")
		return []byte(`{"error": "service required for delete-scope request"}`),
			err
	}

	if !cb.store.HasScope(scope) {
		// this shouldn't fail, just report to the caller
		msg := fmt.Sprintf("{\"error\":\"the scope %s doesn't exist\"}", scope)
		return []byte(msg), nil
	}

	err := cb.store.DeleteScope(scope)
	if err != nil {
		msg := fmt.Sprintf("{\"error\":\"%s\"}", err.Error())
		return []byte(msg), err
	}

	// if the bucket for the scope was successfully removed drop it from the
	// sink list
	cb.manager.RemoveSink(scope)

	return []byte("{}"), nil
}

// Execute callback function to handle `delete` requests.
func (cb *deleteCallback) Execute(msgBody string) ([]byte, error) {
	var (
		scope   string
		key     string
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
		err := errors.New("`service` missing")
		return []byte(`{"error": "service required for delete request"}`), err
	}

	if key, found = request["key"].(string); !found {
		err := errors.New("`key` missing")
		return []byte(`{"error": "key required for delete request"}`), err
	}

	err := cb.store.Delete(scope, key)
	if err != nil {
		msg := fmt.Sprintf("{\"error\":\"%s\"}", err.Error())
		return []byte(msg), err
	}

	return []byte("{}"), nil
}

// Execute callback function to handle `get` requests.
func (cb *getCallback) Execute(msgBody string) ([]byte, error) {
	var (
		scope   string
		key     string
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
		err := errors.New("`service` missing")
		return []byte(`{"error": "service required for get request"}`), err
	}

	if key, found = request["key"].(string); !found {
		err := errors.New("`key` missing")
		return []byte(`{"error": "key required for get request"}`), err
	}

	value, err := cb.store.Get(scope, key)
	if err != nil {
		msg := fmt.Sprintf("{\"error\":\"%s\"}", err.Error())
		return []byte(msg), err
	}

	log.Tracef("value: %s", value)
	return []byte(`{"key": "` + key + `", "value": "` + value + `"}`), nil
}

// Execute callback function to handle `set` requests.
func (cb *setCallback) Execute(msgBody string) ([]byte, error) {
	var (
		scope   string
		key     string
		value   string
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
		err := errors.New("`service` missing")
		return []byte(`{"error": "service required for get request"}`), err
	}

	if key, found = request["key"].(string); !found {
		err := errors.New("`key` missing")
		return []byte(`{"error": "key required for get request"}`), err
	}

	if value, found = request["value"].(string); !found {
		err := errors.New("`value` missing")
		return []byte(`{"error": "value required for get request"}`), err
	}

	err := cb.store.Set(scope, key, value)
	if err != nil {
		msg := fmt.Sprintf("{\"error\":\"%s\"}", err.Error())
		return []byte(msg), err
	}

	log.Tracef("value: %s", value)
	return []byte(`{"key": "` + key + `", "value": "` + value + `"}`), nil
}

// Callback handles subscriber events on the state bus.
func (cb *sinkCallback) Handle(data []byte) error {
	log.WithFields(log.Fields{"data": string(data)}).Debug(
		"data received on state bus")
	return nil
}
