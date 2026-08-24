// Package database opens supported GORM SQL drivers and exposes driver metadata.
//
// MapLoadError and MapPersistError classify GORM errors into D10 categories
// (not_found, conflict, internal) so generated handlers, admin, and auth share
// one unique-violation detector instead of three copies.
package database
