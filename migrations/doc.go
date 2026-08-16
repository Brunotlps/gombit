// Package migrations wraps Atlas versioned migrations for Gombit apps.
//
// MakeMigrations generates SQL via Atlas Program Mode. Migrate, Status, and
// Rollback apply and report those migrations while recording D4 metadata in
// framework_migrations.
package migrations
