package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	conf "github.com/geoffjay/plantd/app/config"
	"github.com/geoffjay/plantd/app/handlers"
	"github.com/geoffjay/plantd/app/internal/auth"
	"github.com/geoffjay/plantd/app/internal/services"
	"github.com/geoffjay/plantd/app/views"
	"github.com/geoffjay/plantd/core/util"

	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/etag"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/template/html/v2"
	log "github.com/sirupsen/logrus"
)

type service struct {
	// Service integrations
	brokerService  *services.BrokerService
	stateService   *services.StateService
	healthService  *services.HealthService
	metricsService *services.MetricsService
}

func (s *service) init() {
	log.WithFields(log.Fields{
		"service": "app",
		"context": "service.init",
	}).Debug("initializing")

	config := conf.GetConfig()

	// Initialize Broker Service
	log.Debug("Initializing Broker Service")
	brokerService, err := services.NewBrokerService(config)
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize Broker Service")
	}
	s.brokerService = brokerService

	// Initialize State Service
	log.Debug("Initializing State Service")
	stateService, err := services.NewStateService(config)
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize State Service")
	}
	s.stateService = stateService

	// Initialize Health Service (depends on broker and state services)
	log.Debug("Initializing Health Service")
	s.healthService = services.NewHealthService(s.brokerService, s.stateService, nil, config)

	// Initialize Metrics Service (depends on broker and state services)
	log.Debug("Initializing Metrics Service")
	s.metricsService = services.NewMetricsService(s.brokerService, s.stateService, config)

	log.WithFields(log.Fields{
		"service": "app",
		"context": "service.init",
	}).Info("All services initialized successfully")
}

func (s *service) run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	log.WithFields(log.Fields{
		"service": "app",
		"context": "service.run",
	}).Debug("starting")

	// Temporarily disable metrics collection to prevent ZeroMQ crashes
	// Start metrics collection
	// if s.metricsService != nil {
	// 	go s.metricsService.StartCollection(ctx)
	// }

	wg.Add(1)
	go s.runApp(ctx, wg)

	<-ctx.Done()

	// Cleanup SSE connections first to prevent goroutine leaks
	CleanupSSEHandler()

	// Stop metrics collection
	if s.metricsService != nil {
		s.metricsService.StopCollection()
	}

	log.WithFields(log.Fields{
		"service": "app",
		"context": "service.run",
	}).Debug("exiting")
}

func (s *service) runApp(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	config := conf.GetConfig()

	fields := log.Fields{"service": "app", "context": "service.run-app"}
	bindAddress := util.Getenv("PLANTD_APP_BIND_ADDRESS", "127.0.0.1")
	bindPort, err := strconv.Atoi(util.Getenv("PLANTD_APP_BIND_PORT", "8443"))
	if err != nil {
		log.WithFields(fields).Fatal(err)
	}

	// Check if HTTP mode is enabled for development
	useHTTP := util.Getenv("PLANTD_APP_USE_HTTP", "false") == "true"
	if useHTTP && config.Env != "production" {
		// Use default HTTP port if not specified and using HTTP
		if util.Getenv("PLANTD_APP_BIND_PORT", "") == "" {
			bindPort = 8080
		}
		log.WithFields(fields).Info("Running in HTTP mode (development only)")
	}

	log.WithFields(fields).Debug("starting server")

	go func() {
		engine := html.New("app/views", ".tmpl")
		engine.Reload(true)
		if config.Env == "development" {
			engine.Debug(true)
		}
		engine.AddFunc("args", views.Args)

		app := fiber.New(fiber.Config{
			Views:       engine,
			JSONEncoder: json.Marshal,
			JSONDecoder: json.Unmarshal,
		})

		// Initialize authentication components
		identityClient, err := auth.NewIdentityClient(config)
		if err != nil {
			log.WithFields(fields).WithError(err).Fatal("Failed to initialize Identity Service client")
		}

		sessionManager, err := auth.NewSessionManager(config, identityClient)
		if err != nil {
			log.WithFields(fields).WithError(err).Fatal("Failed to initialize session manager")
		}

		// Set global session manager for handlers
		handlers.SessionManager = sessionManager

		authHandlers := handlers.NewAuthHandlers(identityClient, sessionManager)
		authMiddleware := auth.NewAuthMiddleware(sessionManager, identityClient)

		// Update health service with identity client now that it's available
		if s.healthService != nil {
			// Pass identity client to health service for health checks
			s.healthService = services.NewHealthService(s.brokerService, s.stateService, identityClient, config)
		}

		sessionStore := session.New(config.Session.ToSessionConfig())
		handlers.SessionStore = sessionStore

		app.Use(helmet.New())
		app.Use(cors.New(config.Cors.ToCorsConfig()))
		app.Use(logger.New())
		app.Use(recover.New())
		app.Use(etag.New())
		app.Use(limiter.New(limiter.Config{
			Expiration: 1 * time.Minute,
			Max:        300,
			KeyGenerator: func(c *fiber.Ctx) string {
				return c.IP()
			},
			LimitReached: func(c *fiber.Ctx) error {
				log.WithFields(log.Fields{
					"service": "app",
					"context": "rate_limiter",
					"ip":      c.IP(),
					"path":    c.Path(),
				}).Warn("Rate limit exceeded")
				return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
					"error": "Too many requests, please try again later",
				})
			},
			SkipFailedRequests:     true,
			SkipSuccessfulRequests: false,
		}))

		// Initialize router with services
		initializeRouter(app, authHandlers, authMiddleware, s, sessionStore)

		address := fmt.Sprintf("%s:%d", bindAddress, bindPort)

		// Start server with or without TLS
		if useHTTP && config.Env != "production" {
			log.WithFields(fields).WithField("address", address).Info("Starting HTTP server (development only)")
			log.WithFields(fields).Fatal(app.Listen(address))
		} else {
			cert := initializeCert()
			tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}

			ln, err := tls.Listen("tcp", address, tlsConfig)
			if err != nil {
				panic(err)
			}

			log.WithFields(fields).WithField("address", address).Info("Starting HTTPS server")
			log.WithFields(fields).Fatal(app.Listener(ln))
		}
	}()

	<-ctx.Done()

	log.WithFields(fields).Debug("exiting server")
}

func initializeCert() tls.Certificate {
	config := conf.GetConfig()
	fields := log.Fields{"service": "app", "context": "service.init-cert"}

	certFile := util.Getenv("PLANTD_APP_TLS_CERT", "cert/app-cert.pem")
	keyFile := util.Getenv("PLANTD_APP_TLS_KEY", "cert/app-key.pem")

	if config.Env == "development" || config.Env == "test" {
		if _, err := os.Stat(certFile); os.IsNotExist(err) {
			log.WithFields(fields).Info(
				"Self-signed certificate not found, generating...")
			if err := generateSelfSignedCert(certFile, keyFile); err != nil {
				log.WithFields(fields).Fatal(err)
			}
			log.WithFields(fields).Info(
				"Self-signed certificate generated successfully")
			log.WithFields(fields).Info(
				"You will need to accept the self-signed certificate " +
					"in your browser")
		}
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.WithFields(fields).Fatal(err)
	}

	return cert
}

// generateSelfSignedCert generates a self-signed certificate and key
// and saves them to the specified files
//
// This is only for testing purposes and should not be used in production.
func generateSelfSignedCert(certFile string, keyFile string) error {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"Plantd Development"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{""},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour * 24 * 365), // 1 year validity

		KeyUsage: x509.KeyUsageKeyEncipherment |
			x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,

		// Add Subject Alternative Names for proper localhost support
		DNSNames: []string{
			"localhost",
			"*.localhost",
			"plantd.local",
			"*.plantd.local",
		},
		IPAddresses: []net.IP{
			net.IPv4(127, 0, 0, 1), // 127.0.0.1
			net.IPv6loopback,       // ::1
		},
	}

	derBytes, err := x509.CreateCertificate(
		rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return err
	}

	certOut, err := os.Create(certFile)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := certOut.Close(); closeErr != nil {
			log.WithFields(log.Fields{
				"service": "app",
				"context": "generateSelfSignedCert",
				"error":   closeErr,
			}).Error("Failed to close certificate file")
		}
	}()

	_ = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	keyOut, err := os.Create(keyFile)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := keyOut.Close(); closeErr != nil {
			log.WithFields(log.Fields{
				"service": "app",
				"context": "generateSelfSignedCert",
				"error":   closeErr,
			}).Error("Failed to close key file")
		}
	}()

	_ = pem.Encode(keyOut, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv),
	})

	return nil
}
