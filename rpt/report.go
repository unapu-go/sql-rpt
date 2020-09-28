package rpt

import (
	"database/sql"
)

type Report struct {
	Schema  Schema
	Finder  Query
	Counter Query

	RawArgs map[string]string `yaml:"raw_args"`
	Declare []map[string]string
	Funcs map[string]string
}

func (this *Report) Count(config *DBConfig, params Paramer) (count uint64, err error) {
	var (
		query string
		args []interface{}
		row *sql.Row
	)
	if query, args, err = this.Counter.Build(config.BindVar, map[string]string{}, params); err != nil {
		return
	}
	row = config.DB.QueryRow(query, args...)
	if err = row.Scan(&count); err != nil {
		return
	}
	return
}

func (this *Report) Find(config *DBConfig, consts map[string]string, params Paramer) (rows *sql.Rows, err error) {
	var (
		query string
		args []interface{}
	)
	if query, args, err = this.Finder.Build(config.BindVar, consts, params); err != nil {
		return
	}
	return config.DB.Query(query, args...)
}
