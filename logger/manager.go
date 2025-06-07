package main

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/geoffjay/plantd/core/bus"
	"github.com/geoffjay/plantd/core/util"
	"github.com/geoffjay/plantd/logger/db"
	"github.com/geoffjay/plantd/logger/db/migrations"

	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

// Manager handles database operations and message bus subscriptions for the
// logger service.
type Manager struct {
	db         *sql.DB
	migrations []db.Migration

	// bus subscribers
	stateSink  *bus.Sink
	eventSink  *bus.Sink
	metricSink *bus.Sink
}

// NewManager constructs and instance of a manager type.
func NewManager() *Manager {
	return &Manager{
		migrations: make([]db.Migration, 0),
	}
}

// Run starts the manager with database connection, migrations, and message bus
// subscriptions.
func (m *Manager) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	log.WithFields(log.Fields{"context": "manager.run"}).Debug("starting")

	m.openDB()
	defer func() {
		if closeErr := m.db.Close(); closeErr != nil {
			log.WithFields(log.Fields{
				"context": "manager.run",
				"error":   closeErr,
			}).Error("failed to close database connection")
		}
	}()
	m.runMigrations()

	m.stateSink = bus.NewSink(">tcp://localhost:11001", "org.plantd")
	m.eventSink = bus.NewSink(">tcp://localhost:12001", "org.plantd")
	m.metricSink = bus.NewSink(">tcp://localhost:13001", "org.plantd")
	defer m.stateSink.Stop()
	defer m.eventSink.Stop()
	defer m.metricSink.Stop()

	m.stateSink.SetHandler(&bus.SinkHandler{
		Callback: &stateSinkCallback{db: m.db}})
	m.eventSink.SetHandler(&bus.SinkHandler{
		Callback: &eventSinkCallback{db: m.db}})
	m.metricSink.SetHandler(&bus.SinkHandler{
		Callback: &metricSinkCallback{db: m.db}})

	wg.Add(3)
	go m.eventSink.Run(ctx, wg)
	go m.metricSink.Run(ctx, wg)
	go m.stateSink.Run(ctx, wg)

	<-ctx.Done()

	log.WithFields(log.Fields{"context": "manager.run"}).Debug("exiting")
}

func connectionStr() string {
	host := util.Getenv("PLANTD_LOGGER_TSDB_HOST", "localhost")
	port := util.Getenv("PLANTD_LOGGER_TSDB_PORT", "5432")
	user := util.Getenv("PLANTD_LOGGER_TSDB_USER", "admin")
	password := util.Getenv("PLANTD_LOGGER_TSDB_PASSWORD", "plantd")
	database := util.Getenv("PLANTD_LOGGER_TSDB_DATABASE", "plantd_development")

	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host,
		port,
		user,
		password,
		database,
	)
}

func (m *Manager) openDB() {
	var err error

	log.Debug("connecting to database")

	m.db, err = sql.Open("postgres", connectionStr())
	if err != nil {
		log.Panic(err)
	}
}

func (m *Manager) runMigrations() {
	log.Debug("loading database migrations")
	m.registerMigration(&migrations.CreateMetricsTable{DB: m.db})

	log.Debug("running database migrations")

	// XXX: should these rollback if one fails?
	for _, migration := range m.migrations {
		if err := migration.Up(); err != nil {
			log.Panic(err)
		}
	}
}

func (m *Manager) registerMigration(migration db.Migration) {
	m.migrations = append(m.migrations, migration)
}
