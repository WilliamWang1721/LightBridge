package modulemigration

import (
	"context"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestHasLegacyOpenAIAccountsExcludesAlreadyMigratedRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New(): %v", err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(regexp.QuoteMeta(`
SELECT COUNT(*)
FROM accounts
WHERE deleted_at IS NULL
  AND platform = 'openai'
  AND COALESCE(extra->>'provider_id', '') <> 'openai'
  AND COALESCE(extra->'module_migration'->>'provider_id', '') <> 'openai'`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	if hasLegacyOpenAIAccounts(context.Background(), db) {
		t.Fatal("hasLegacyOpenAIAccounts() = true, want false for already-migrated rows")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestHasLegacyOpenAIAccountsFindsUnmarkedRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New(): %v", err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT COUNT\\(\\*\\)[\\s\\S]+platform = 'openai'[\\s\\S]+extra->>'provider_id'").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	if !hasLegacyOpenAIAccounts(context.Background(), db) {
		t.Fatal("hasLegacyOpenAIAccounts() = false, want true for unmarked OpenAI rows")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestHasOpenAIProviderModuleAccountsFindsTopLevelProviderMarker(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(regexp.QuoteMeta(`
SELECT COUNT(*)
FROM accounts
WHERE deleted_at IS NULL
  AND (
    LOWER(BTRIM(COALESCE(extra->>'provider_id', ''))) = 'openai'
    OR LOWER(BTRIM(COALESCE(extra->'module_migration'->>'provider_id', ''))) = 'openai'
  )`)).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	if !hasOpenAIProviderModuleAccounts(context.Background(), db) {
		t.Fatal("provider-marked OpenAI account should require a missing module to be recovered")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestHasOpenAIProviderModuleAccountsIgnoresUnboundAccounts(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT COUNT\\(\\*\\).*FROM accounts").WillReturnRows(
		sqlmock.NewRows([]string{"count"}).AddRow(0),
	)
	if hasOpenAIProviderModuleAccounts(context.Background(), db) {
		t.Fatal("accounts without an explicit OpenAI provider marker must not trigger module recovery")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
