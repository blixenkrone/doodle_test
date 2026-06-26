package postgres

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrForeignKeyConstraintViolation = errors.New("foreign key constraint violation")
	ErrUniqueConstraintViolation     = errors.New("unique constraint violation")
	ErrCheckConstraintViolation      = errors.New("check constraint violation")

	// Returned from a QueryRow operation
	ErrNoRows = pgx.ErrNoRows
)

const (
	ForeignConstraintPgErrCode = "23503"
	UniqueConstraintPgErrCode  = "23505"
	CheckConstraintPgErrCode   = "23514"
)

func WrapPostgresError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch code := pgErr.Code; code {
		case ForeignConstraintPgErrCode:
			return fmt.Errorf("driver error: %v - mapped: %w", err, ErrForeignKeyConstraintViolation)
		case UniqueConstraintPgErrCode:
			return fmt.Errorf("driver error: %v - mapped: %w", err, ErrUniqueConstraintViolation)
		case CheckConstraintPgErrCode:
			return fmt.Errorf("driver error: %v - mapped: %w", err, ErrCheckConstraintViolation)
		default:
			return fmt.Errorf("unhandled postgres error (code %q): %w", code, err)
		}
	}
	return err // Return the OG error - we dont know what it is

}

func IsUniqueConstraintErr(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == UniqueConstraintPgErrCode
	}
	return false
}

func IsCheckConstraintErr(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == CheckConstraintPgErrCode
	}
	return false
}

func IsForeignKeyConstraintErr(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == ForeignConstraintPgErrCode
	}
	return false
}
