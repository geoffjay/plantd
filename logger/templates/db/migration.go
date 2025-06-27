package migrations

import (
	"context"
	"database/sql"
)

type MigrationName struct {
	DB *sql.DB
}

func (m *MigrationName) Up() error {
	sql := `-- up query`

	ctx := context.Background()
	_, err := m.DB.ExecContext(ctx, sql)
	if err != nil {
		return err
	}

	return nil
}

func (m *MigrationName) Down() error {
	sql := `-- down query`

	ctx := context.Background()
	_, err := m.DB.ExecContext(ctx, sql)
	if err != nil {
		return err
	}

	return nil
}
