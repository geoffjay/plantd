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
	"os"
	"strconv"
	"sync"
	"time"

	conf "github.com/geoffjay/plantd/app/config"
	"github.com/geoffjay/plantd/app/handlers"
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

type service struct{}

func (s *service) init() {
	log.WithFields(log.Fields{
		"service": "app",
		"context": "service.init",
	}).Debug("initializing")

	// TODO: Initialize Identity Service client and other service integrations
	// Repository initialization removed - using Identity Service for user management
}

func (s *service) run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	log.WithFields(log.Fields{
		"service": "app",
		"context": "service.run",
	}).Debug("starting")

	wg.Add(1)
	go s.runApp(ctx, wg)

	<-ctx.Done()

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

		handlers.SessionStore = session.New(config.Session.ToSessionConfig())

		app.Use(helmet.New())
		app.Use(cors.New(config.Cors.ToCorsConfig()))
		app.Use(logger.New())
		app.Use(recover.New())
		app.Use(etag.New())
		app.Use(limiter.New(limiter.Config{
			Expiration: 30 * time.Second,
			Max:        50,
		}))

		initializeRouter(app)

		cert := initializeCert()
		tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
		address := fmt.Sprintf("%s:%d", bindAddress, bindPort)

		ln, err := tls.Listen("tcp", address, tlsConfig)
		if err != nil {
			panic(err)
		}

		log.WithFields(fields).Fatal(app.Listener(ln))
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
			Organization: []string{"Plantd Org"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour * 24 * 180),

		KeyUsage: x509.KeyUsageKeyEncipherment |
			x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
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
