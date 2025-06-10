package handlers

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBaseHandler(t *testing.T) {
	logger := logrus.New()
	handler := NewBaseHandler("test-service", logger)

	assert.NotNil(t, handler)
	assert.Equal(t, "test-service", handler.GetServiceName())
	assert.NotNil(t, handler.validator)
	assert.Equal(t, logger, handler.logger)
}

func TestBaseHandler_ParseRequest(t *testing.T) {
	logger := logrus.New()
	handler := NewBaseHandler("test", logger)

	type TestRequest struct {
		Name  string `json:"name" validate:"required"`
		Email string `json:"email" validate:"required,email"`
	}

	tests := []struct {
		name    string
		payload []byte
		wantErr bool
	}{
		{
			name:    "valid request",
			payload: []byte(`{"name": "John Doe", "email": "john@example.com"}`),
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			payload: []byte(`{"name": "John`),
			wantErr: true,
		},
		{
			name:    "missing required field",
			payload: []byte(`{"name": "John Doe"}`),
			wantErr: true,
		},
		{
			name:    "invalid email format",
			payload: []byte(`{"name": "John Doe", "email": "invalid-email"}`),
			wantErr: true,
		},
		{
			name:    "empty payload",
			payload: []byte(``),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req TestRequest
			err := handler.ParseRequest(tt.payload, &req)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "John Doe", req.Name)
				assert.Equal(t, "john@example.com", req.Email)
			}
		})
	}
}

func TestBaseHandler_CreateSuccessResponse(t *testing.T) {
	logger := logrus.New()
	handler := NewBaseHandler("test", logger)

	tests := []struct {
		name      string
		requestID string
		data      interface{}
		wantErr   bool
	}{
		{
			name:      "success response with data",
			requestID: "test-123",
			data:      map[string]string{"user_id": "123"},
			wantErr:   false,
		},
		{
			name:      "success response without data",
			requestID: "test-456",
			data:      nil,
			wantErr:   false,
		},
		{
			name:      "success response with complex data",
			requestID: "test-789",
			data: LoginResponse{
				Header: ResponseHeader{
					RequestID: "inner-123",
					Success:   true,
					Timestamp: time.Now().Unix(),
				},
				User: nil,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := handler.CreateSuccessResponse(tt.requestID, tt.data)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)

				// Parse the JSON response
				var result map[string]interface{}
				err := json.Unmarshal(response, &result)
				require.NoError(t, err)

				// Verify header structure
				header, ok := result["header"].(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, tt.requestID, header["request_id"])
				assert.Equal(t, true, header["success"])
				assert.NotNil(t, header["timestamp"])
			}
		})
	}
}

func TestBaseHandler_CreateErrorResponse(t *testing.T) {
	logger := logrus.New()
	handler := NewBaseHandler("test", logger)

	tests := []struct {
		name         string
		requestID    string
		errorCode    string
		errorMessage string
		detail       string
		wantErr      bool
	}{
		{
			name:         "basic error response",
			requestID:    "test-123",
			errorCode:    "VALIDATION_ERROR",
			errorMessage: "Invalid request",
			detail:       "Field 'email' is required",
			wantErr:      false,
		},
		{
			name:         "error response without detail",
			requestID:    "test-456",
			errorCode:    "NOT_FOUND",
			errorMessage: "User not found",
			detail:       "",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := handler.CreateErrorResponse(tt.requestID, tt.errorCode, tt.errorMessage, tt.detail)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)

				// Parse the JSON response
				var result ErrorResponse
				err := json.Unmarshal(response, &result)
				require.NoError(t, err)

				// Verify error response structure
				assert.Equal(t, tt.requestID, result.Header.RequestID)
				assert.False(t, result.Header.Success)
				assert.Equal(t, tt.errorMessage, result.Header.Error)
				assert.Equal(t, tt.errorCode, result.Code)
				assert.Equal(t, tt.detail, result.Detail)
				assert.NotZero(t, result.Header.Timestamp)
			}
		})
	}
}

func TestBaseHandler_ExtractRequestID(t *testing.T) {
	logger := logrus.New()
	handler := NewBaseHandler("test", logger)

	tests := []struct {
		name     string
		req      interface{}
		expected string
	}{
		{
			name: "login request",
			req: &LoginRequest{
				Header: RequestHeader{RequestID: "login-123"},
			},
			expected: "login-123",
		},
		{
			name: "create user request",
			req: &CreateUserRequest{
				Header: RequestHeader{RequestID: "user-456"},
			},
			expected: "user-456",
		},
		{
			name: "validate token request",
			req: &ValidateTokenRequest{
				Header: RequestHeader{RequestID: "token-789"},
			},
			expected: "token-789",
		},
		{
			name:     "unknown request type",
			req:      "invalid",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestID := handler.ExtractRequestID(tt.req)
			assert.Equal(t, tt.expected, requestID)
		})
	}
}

func TestBaseHandler_ExtractUserID(t *testing.T) {
	logger := logrus.New()
	handler := NewBaseHandler("test", logger)

	userID1 := uint(123)
	userID2 := uint(456)

	tests := []struct {
		name     string
		req      interface{}
		expected *uint
	}{
		{
			name: "request with user ID",
			req: &LoginRequest{
				Header: RequestHeader{UserID: &userID1},
			},
			expected: &userID1,
		},
		{
			name: "request with different user ID",
			req: &UpdateUserRequest{
				Header: RequestHeader{UserID: &userID2},
			},
			expected: &userID2,
		},
		{
			name: "request without user ID",
			req: &LoginRequest{
				Header: RequestHeader{UserID: nil},
			},
			expected: nil,
		},
		{
			name:     "unknown request type",
			req:      "invalid",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID := handler.ExtractUserID(tt.req)
			if tt.expected == nil {
				assert.Nil(t, userID)
			} else {
				require.NotNil(t, userID)
				assert.Equal(t, *tt.expected, *userID)
			}
		})
	}
}

func TestBaseHandler_LogRequest(t *testing.T) {
	logger := logrus.New()
	handler := NewBaseHandler("test-service", logger)

	userID := uint(123)

	tests := []struct {
		name      string
		operation string
		requestID string
		userID    *uint
	}{
		{
			name:      "log request with user ID",
			operation: "LOGIN",
			requestID: "req-123",
			userID:    &userID,
		},
		{
			name:      "log request without user ID",
			operation: "HEALTH_CHECK",
			requestID: "req-456",
			userID:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that LogRequest doesn't panic
			assert.NotPanics(t, func() {
				handler.LogRequest(tt.operation, tt.requestID, tt.userID)
			})
		})
	}
}

func TestBaseHandler_LogResponse(t *testing.T) {
	logger := logrus.New()
	handler := NewBaseHandler("test-service", logger)

	tests := []struct {
		name      string
		operation string
		requestID string
		success   bool
		err       error
	}{
		{
			name:      "log successful response",
			operation: "LOGIN",
			requestID: "req-123",
			success:   true,
			err:       nil,
		},
		{
			name:      "log error response",
			operation: "LOGIN",
			requestID: "req-456",
			success:   false,
			err:       assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that LogResponse doesn't panic
			assert.NotPanics(t, func() {
				handler.LogResponse(tt.operation, tt.requestID, tt.success, tt.err)
			})
		})
	}
}

func TestBaseHandler_HandlePanic(t *testing.T) {
	logger := logrus.New()
	handler := NewBaseHandler("test-service", logger)

	t.Run("no panic", func(t *testing.T) {
		response, err := handler.HandlePanic("test-request-123")

		assert.NoError(t, err)
		assert.Nil(t, response) // Should be nil when no panic occurs
	})

	t.Run("test error response creation", func(t *testing.T) {
		// Test the error response creation functionality directly
		response, err := handler.CreateErrorResponse("test-request-123", "INTERNAL_ERROR", "Internal server error", "Panic: test panic")

		assert.NoError(t, err)
		assert.NotNil(t, response)

		// Parse the response to verify it's a proper error response
		var result ErrorResponse
		err = json.Unmarshal(response, &result)
		require.NoError(t, err)

		assert.Equal(t, "test-request-123", result.Header.RequestID)
		assert.False(t, result.Header.Success)
		assert.Equal(t, "INTERNAL_ERROR", result.Code)
		assert.Contains(t, result.Header.Error, "Internal server error")
		assert.Contains(t, result.Detail, "Panic: test panic")
	})
}
