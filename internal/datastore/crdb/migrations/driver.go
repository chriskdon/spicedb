package migrations

import (
	"context"
	"errors"
	"fmt"

	"github.com/authzed/spicedb/pkg/migrate"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/log/zerologadapter"
	"github.com/rs/zerolog/log"
)

const (
	errUnableToInstantiate = "unable to instantiate CRDBDriver: %w"

	postgresMissingTableErrorCode = "42P01"

	queryLoadVersion  = "SELECT version_num from schema_version"
	queryWriteVersion = "UPDATE schema_version SET version_num=$1 WHERE version_num=$2"
)

// CRDBDriver implements a schema migration facility for use in SpiceDB's CRDB
// datastore.
type CRDBDriver struct {
	db *pgx.Conn
}

// NewCRDBDriver creates a new driver with active connections to the database
// specified.
func NewCRDBDriver(url string) (*CRDBDriver, error) {
	connConfig, err := pgx.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf(errUnableToInstantiate, err)
	}

	connConfig.Logger = zerologadapter.NewLogger(log.Logger)

	db, err := pgx.ConnectConfig(context.Background(), connConfig)
	if err != nil {
		return nil, fmt.Errorf(errUnableToInstantiate, err)
	}

	return &CRDBDriver{db}, nil
}

// Version returns the version of the schema to which the connected database
// has been migrated.
func (apd *CRDBDriver) Version(ctx context.Context) (string, error) {
	var loaded string

	if err := apd.db.QueryRow(ctx, queryLoadVersion).Scan(&loaded); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == postgresMissingTableErrorCode {
			return "", nil
		}
		return "", fmt.Errorf("unable to load alembic revision: %w", err)
	}

	return loaded, nil
}

func (apd *CRDBDriver) Transact(ctx context.Context, f migrate.MigrationFunc[pgx.Tx], version, replaced string) error {
	tx, err := apd.db.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadWrite})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	err = f(ctx, tx)
	if err != nil {
		return err
	}
	err = writeVersion(ctx, tx, version, replaced)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// WriteVersion overwrites the value stored to track the version of the
// database schema.
func writeVersion(ctx context.Context, tx pgx.Tx, version, replaced string) error {
	result, err := tx.Exec(ctx, queryWriteVersion, version, replaced)
	if err != nil {
		return fmt.Errorf("unable to update version row: %w", err)
	}

	updatedCount := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("unable to compute number of rows affected: %w", err)
	}

	if updatedCount != 1 {
		return fmt.Errorf("writing version update affected %d rows, should be 1", updatedCount)
	}

	return nil
}

// Close disposes the driver.
func (apd *CRDBDriver) Close(ctx context.Context) error {
	return apd.db.Close(ctx)
}
