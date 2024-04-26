package store

import _ "embed"

// Schema for creating SQLite table
//
//go:embed schema.sql
var SchemaSQL string
