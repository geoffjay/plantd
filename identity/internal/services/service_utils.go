package services

import (
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"
)

// Common field names used across services.
const (
	FieldName        = "name"
	FieldSlug        = "slug"
	FieldEmail       = "email"
	FieldUsername    = "username"
	FieldDescription = "description"
)

// Common error messages.
const (
	ErrNotFound      = "not found"
	ErrAlreadyExists = "already exists"
	ErrEmptyField    = "cannot be empty"
)

// logAndError creates a standardized error with logging.
func logAndError(logger *log.Entry, message string, err error) error {
	logger.WithError(err).Error(message)
	return fmt.Errorf("%s: %w", message, err)
}

// logAndErrorSimple creates a standardized error with logging for simple messages.
func logAndErrorSimple(logger *log.Entry, message string) error {
	logger.Error(message)
	return errors.New(message)
}

// logSuccess logs a successful operation.
func logSuccess(logger *log.Entry, message string, fields log.Fields) {
	if fields != nil {
		logger = logger.WithFields(fields)
	}
	logger.Info(message)
}

// createServiceLogger creates a standardized logger for service methods.
func createServiceLogger(serviceName, methodName string, fields log.Fields) *log.Entry {
	logFields := log.Fields{
		"service": serviceName,
		"method":  methodName,
	}

	// Merge additional fields
	for k, v := range fields {
		logFields[k] = v
	}

	return log.WithFields(logFields)
}
