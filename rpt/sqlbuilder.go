package rpt

import (
	"io"
	"os"
	"strconv"

	"github.com/pkg/errors"
)

type SqlBuilder struct {
	Path    string
	rpt     *Report
	Params  map[string][]string
	BindVar func(i int) string
}

func (this *SqlBuilder) Report() *Report {
	return this.rpt
}

func (this *SqlBuilder) Build() (err error) {
	var r io.Reader
	if this.Path == "-" {
		r = os.Stdin
	} else {
		var f *os.File
		if f, err = os.Open(this.Path); err != nil {
			return
		}
		defer f.Close()
		r = f
	}
	if this.rpt, err = Load(r); err != nil {
		return
	}
	return
}

func (this *SqlBuilder) Counter() (query string, args []interface{}, err error) {
	return this.rpt.Counter.Build(this.BindVar, map[string]string{}, UrlParams{this.Params})
}

func (this *SqlBuilder) Finder() (query string, args []interface{}, err error) {
	var page, skip, perPage uint64
	perPage = PerPage
	params := UrlParams{this.Params}
	if pageS, _ := params.Value(":page"); pageS != "" {
		if page, err = strconv.ParseUint(pageS, 10, 64); err != nil {
			err = errors.Wrap(err, "parse `:page`:")
			return
		}
	}
	if perPageS, _ := params.Value(":per_page"); perPageS != "" {
		if perPage, err = strconv.ParseUint(perPageS, 10, 64); err != nil {
			err = errors.Wrap(err, "parse `:per_page`:")
			return
		}
		if perPage > PerPage {
			err = errors.New("`:per_page` value is greather than 500 records")
			return
		}
	}
	if page == 0 {
		page = 1
	}

	if page > 1 {
		skip = (page - 1) * perPage
	}

	return this.rpt.Finder.Build(this.BindVar, map[string]string{
		"OFFSET": strconv.FormatUint(skip, 10),
		"LIMIT":  strconv.FormatUint(perPage, 10),
	}, params)
}
