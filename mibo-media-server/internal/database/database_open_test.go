package database

import "testing"

func TestDatabaseOpenFreshCatalogDatabaseMigratesLegacyAndCatalogTables(t *testing.T) {
	dbPath := newCatalogDatabasePath(t)
	db := openDatabaseForMigrationAssertions(t, dbPath)

	assertTablesExist(t, db, requiredFreshStartupModels())
	assertCatalogIndexesExist(t, db)
}

func TestDatabaseOpenMigratesLegacyOnlyCatalogDatabase(t *testing.T) {
	dbPath := newCatalogDatabasePath(t)
	seedLegacyOnlyDatabase(t, dbPath)

	db := openDatabaseForMigrationAssertions(t, dbPath)
	assertTablesExist(t, db, requiredFreshStartupModels())
	assertLegacyRowsPreserved(t, db)
	assertCatalogIndexesExist(t, db)
}

func TestDatabaseOpenCatalogMigrationIsIdempotent(t *testing.T) {
	dbPath := newCatalogDatabasePath(t)
	seedLegacyOnlyDatabase(t, dbPath)

	first := openDatabaseForMigrationAssertions(t, dbPath)
	assertCatalogIndexesExist(t, first)

	second := openDatabaseForMigrationAssertions(t, dbPath)
	assertTablesExist(t, second, requiredFreshStartupModels())
	assertLegacyRowsPreserved(t, second)
	assertCatalogIndexesExist(t, second)
}
