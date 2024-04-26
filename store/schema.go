package store

// Schema for creating SQLite table
const Schema = `
CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	email TEXT,
	password TEXT,
	name TEXT,
	gender TEXT,
	dob TEXT
);

`
