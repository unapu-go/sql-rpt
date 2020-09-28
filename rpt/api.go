package rpt

import "database/sql"

type Paramer interface {
	Values(name string) ([]string, bool)
	Value(name string) (string,bool)
}

type DB interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}