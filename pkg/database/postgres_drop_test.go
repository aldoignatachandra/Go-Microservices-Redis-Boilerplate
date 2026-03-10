package database

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newMockGormDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn:                 sqlDB,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	return gormDB, mock
}

func TestDropDatabaseAndVerify_ReturnsErrorWhenDropFails(t *testing.T) {
	db, mock := newMockGormDB(t)
	ctx := context.Background()

	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()")).
		WithArgs("app_db").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`DROP DATABASE "app_db"`)).
		WillReturnError(assertionErr("database is being accessed by other users"))

	err := dropDatabaseAndVerify(ctx, db, "app_db")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to drop database")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDropDatabaseAndVerify_ReturnsErrorWhenDatabaseStillExists(t *testing.T) {
	db, mock := newMockGormDB(t)
	ctx := context.Background()

	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()")).
		WithArgs("app_db").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`DROP DATABASE "app_db"`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM pg_database WHERE datname = $1")).
		WithArgs("app_db").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	err := dropDatabaseAndVerify(ctx, db, "app_db")
	require.Error(t, err)
	require.Contains(t, err.Error(), "still exists")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDropDatabaseAndVerify_Success(t *testing.T) {
	db, mock := newMockGormDB(t)
	ctx := context.Background()

	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()")).
		WithArgs("app_db").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`DROP DATABASE "app_db"`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM pg_database WHERE datname = $1")).
		WithArgs("app_db").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	err := dropDatabaseAndVerify(ctx, db, "app_db")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateDatabaseAndVerify_ReturnsErrorWhenCreateFails(t *testing.T) {
	db, mock := newMockGormDB(t)
	ctx := context.Background()

	mock.ExpectExec(regexp.QuoteMeta(`CREATE DATABASE "app_db"`)).
		WillReturnError(assertionErr("permission denied to create database"))

	err := createDatabaseAndVerify(ctx, db, "app_db")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create database")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateDatabaseAndVerify_ReturnsErrorWhenDatabaseStillMissing(t *testing.T) {
	db, mock := newMockGormDB(t)
	ctx := context.Background()

	mock.ExpectExec(regexp.QuoteMeta(`CREATE DATABASE "app_db"`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM pg_database WHERE datname = $1")).
		WithArgs("app_db").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	err := createDatabaseAndVerify(ctx, db, "app_db")
	require.Error(t, err)
	require.Contains(t, err.Error(), "still does not exist")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateDatabaseAndVerify_Success(t *testing.T) {
	db, mock := newMockGormDB(t)
	ctx := context.Background()

	mock.ExpectExec(regexp.QuoteMeta(`CREATE DATABASE "app_db"`)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM pg_database WHERE datname = $1")).
		WithArgs("app_db").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	err := createDatabaseAndVerify(ctx, db, "app_db")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestQuoteIdentifier_EscapesQuotes(t *testing.T) {
	got := quoteIdentifier(`my"db`)
	require.Equal(t, `"my""db"`, got)
}

type assertionErr string

func (e assertionErr) Error() string { return string(e) }
