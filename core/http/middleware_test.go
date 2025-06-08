package http

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func setupMiddlewareTest() (bytes.Buffer, func()) {
	gin.SetMode(gin.TestMode)

	var logOutput bytes.Buffer
	originalOutput := log.StandardLogger().Out
	log.SetOutput(&logOutput)

	originalLevel := log.GetLevel()
	log.SetLevel(log.InfoLevel)

	cleanup := func() {
		log.SetOutput(originalOutput)
		log.SetLevel(originalLevel)
	}

	return logOutput, cleanup
}

func TestLoggerMiddlewareBasic(t *testing.T) {
	logOutput, cleanup := setupMiddlewareTest()
	defer cleanup()

	// Create gin router with middleware
	router := gin.New()
	router.Use(LoggerMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.1")
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	// Check that log was written
	logString := logOutput.String()
	assert.Contains(t, logString, "GET")
	assert.Contains(t, logString, "/test")
	assert.Contains(t, logString, "200")
	assert.Contains(t, logString, "status")
	assert.Contains(t, logString, "latency")
	assert.Contains(t, logString, "client_ip")
	assert.Contains(t, logString, "req_method")
	assert.Contains(t, logString, "req_uri")
}

func TestLoggerMiddlewareHTTPMethods(t *testing.T) {
	logOutput, cleanup := setupMiddlewareTest()
	defer cleanup()

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			// Clear log output
			logOutput.Reset()

			// Create gin router with middleware
			router := gin.New()
			router.Use(LoggerMiddleware())
			router.Handle(method, "/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"method": method})
			})

			// Create test request
			req := httptest.NewRequest(method, "/test", nil)
			w := httptest.NewRecorder()

			// Execute request
			router.ServeHTTP(w, req)

			// Check that method is logged
			logString := logOutput.String()
			assert.Contains(t, logString, method)
		})
	}
}

func TestLoggerMiddlewareStatusCodes(t *testing.T) {
	logOutput, cleanup := setupMiddlewareTest()
	defer cleanup()

	statusCodes := []int{200, 201, 400, 404, 500}

	for _, statusCode := range statusCodes {
		t.Run(fmt.Sprintf("status_%d", statusCode), func(t *testing.T) {
			// Clear log output
			logOutput.Reset()

			// Create gin router with middleware
			router := gin.New()
			router.Use(LoggerMiddleware())
			router.GET("/test", func(c *gin.Context) {
				c.JSON(statusCode, gin.H{"status": statusCode})
			})

			// Create test request
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()

			// Execute request
			router.ServeHTTP(w, req)

			// Check that status code is logged
			logString := logOutput.String()
			assert.Contains(t, logString, fmt.Sprintf("status=%d", statusCode))
		})
	}
}

func TestLoggerMiddlewareLatency(t *testing.T) {
	logOutput, cleanup := setupMiddlewareTest()
	defer cleanup()

	// Clear log output
	logOutput.Reset()

	// Create gin router with middleware
	router := gin.New()
	router.Use(LoggerMiddleware())
	router.GET("/slow", func(c *gin.Context) {
		// Simulate slow endpoint
		time.Sleep(10 * time.Millisecond)
		c.JSON(http.StatusOK, gin.H{"message": "slow"})
	})

	// Create test request
	req := httptest.NewRequest("GET", "/slow", nil)
	w := httptest.NewRecorder()

	// Execute request
	start := time.Now()
	router.ServeHTTP(w, req)
	actualDuration := time.Since(start)

	// Check that latency is logged and reasonable
	logString := logOutput.String()
	assert.Contains(t, logString, "latency")

	// The actual duration should be at least 10ms
	assert.GreaterOrEqual(t, actualDuration, 10*time.Millisecond)
}

func TestLoggerMiddlewareClientIP(t *testing.T) {
	logOutput, cleanup := setupMiddlewareTest()
	defer cleanup()

	testCases := []struct {
		name          string
		remoteAddr    string
		xForwardedFor string
		xRealIP       string
		expectedInLog string
	}{
		{
			name:          "direct connection",
			remoteAddr:    "192.168.1.100:12345",
			expectedInLog: "192.168.1.100",
		},
		{
			name:          "x-forwarded-for header",
			remoteAddr:    "127.0.0.1:12345",
			xForwardedFor: "203.0.113.1",
			expectedInLog: "203.0.113.1",
		},
		{
			name:          "x-real-ip header",
			remoteAddr:    "127.0.0.1:12345",
			xRealIP:       "203.0.113.2",
			expectedInLog: "203.0.113.2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clear log output
			logOutput.Reset()

			// Create gin router with middleware
			router := gin.New()
			router.Use(LoggerMiddleware())
			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "test"})
			})

			// Create test request
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tc.remoteAddr
			if tc.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tc.xForwardedFor)
			}
			if tc.xRealIP != "" {
				req.Header.Set("X-Real-IP", tc.xRealIP)
			}

			w := httptest.NewRecorder()

			// Execute request
			router.ServeHTTP(w, req)

			// Check that expected IP is logged
			logString := logOutput.String()
			assert.Contains(t, logString, tc.expectedInLog)
		})
	}
}

func TestLoggerMiddlewareQueryParameters(t *testing.T) {
	logOutput, cleanup := setupMiddlewareTest()
	defer cleanup()

	// Clear log output
	logOutput.Reset()

	// Create gin router with middleware
	router := gin.New()
	router.Use(LoggerMiddleware())
	router.GET("/search", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"query": c.Query("q")})
	})

	// Create test request with query parameters
	req := httptest.NewRequest("GET", "/search?q=test&limit=10", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Check that full URI with query params is logged
	logString := logOutput.String()
	assert.Contains(t, logString, "/search?q=test&limit=10")
}

func TestLoggerMiddlewareContinuesToNext(t *testing.T) {
	_, cleanup := setupMiddlewareTest()
	defer cleanup()

	handlerCalled := false

	// Create gin router with middleware
	router := gin.New()
	router.Use(LoggerMiddleware())
	router.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Check that the actual handler was called
	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, w.Code)
}
