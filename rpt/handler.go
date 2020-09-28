package rpt

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"

	"github.com/moisespsena-go/httpu"

	"github.com/ecletus/core/utils"
	"github.com/moisespsena-go/bintb"
)

const PerPage uint64 = 500

type Handler func(w http.ResponseWriter, r *http.Request)

type Handlers struct {
	RequestDada func(r *http.Request) (name string, db *DBConfig, sources []string)
}

func (this Handlers) GetRpt(w http.ResponseWriter, name string, sources []string) *Report {
	rpt, err := GetReport(name, sources...)
	if err == nil {
		_, err = rpt.Schema.Columns()
	}
	if err != nil {
		if err == os.ErrNotExist {
			http.Error(w, "report does not exists", http.StatusNotFound)
			return nil
		}
		http.Error(w, "load report failed: "+err.Error(), http.StatusUnprocessableEntity)
		return nil
	}
	return rpt
}

func (this Handlers) Schema(w http.ResponseWriter, r *http.Request) {
	name, _, sources := this.RequestDada(r)
	rpt := this.GetRpt(w, name, sources)
	if rpt == nil {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	var bw bytes.Buffer
	if err := json.NewEncoder(&bw).Encode(rpt.Schema); err != nil {
		http.Error(w, "encode schema failed: "+err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.Write(bw.Bytes())
}

func (this Handlers) Stat(w http.ResponseWriter, r *http.Request) {
	name, db, sources := this.RequestDada(r)
	rpt := this.GetRpt(w, name, sources)
	if rpt == nil {
		return
	}
	count, err := rpt.Count(db, UrlParams{r.URL.Query()})
	if err != nil {
		http.Error(w, "report does not exists", http.StatusUnprocessableEntity)
		return
	}
	var perPage = PerPage

	if perPageS := r.URL.Query().Get(":per_page"); perPageS != "" {
		var err error
		if perPage, err = strconv.ParseUint(perPageS, 10, 64); err != nil {
			http.Error(w, "parse `:per_page`: "+err.Error(), http.StatusUnprocessableEntity)
			return
		}
		if perPage > PerPage {
			http.Error(w, "`:per_page` value is greather than 500 records", http.StatusPreconditionFailed)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	pages := (count-1)/perPage + 1
	var bw bytes.Buffer
	json.NewEncoder(&bw).Encode(map[string]interface{}{
		"total":    count,
		"pages":    pages,
		"per_page": perPage,
	})
	w.Write(bw.Bytes())
}

func (this Handlers) DataJson(w io.Writer, onlyRecords, spec bool, columns []*bintb.Column, next func() (_ bintb.Recorde, err error)) error {
	var enc = json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return bintb.JsonStreamWriteF(&bintb.JsonStreamWriterConfig{
		JsonWriterConfig: bintb.JsonWriterConfig{
			Encoder:      bintb.NewEncoder(columns, context.Background()),
			OnlyNames:    !spec,
			Arrays:       true,
			JosonEncoder: enc,
		},
		OnlyRecords: onlyRecords,
	}, next)
}
func (this Handlers) DataCsv(w io.Writer, onlyRecords, spec bool, columns []*bintb.Column, next func() (_ bintb.Recorde, err error)) error {
	return bintb.CsvStreamWriteF(bintb.CsvStreamWriterOptions{
		Encoder:         bintb.NewEncoder(columns, context.Background()),
		OnlyColumnNames: !onlyRecords && !spec,
		OnlyRecords:     onlyRecords,
	}, csv.NewWriter(w), next)
}

func (this Handlers) Data(rw http.ResponseWriter, r *http.Request) {
	w := httpu.ResponseWriterOf(rw)
	name, db, sources := this.RequestDada(r)
	rpt := this.GetRpt(w, name, sources)
	if rpt == nil {
		return
	}
	var (
		err               error
		query             = r.URL.Query()
		urlp              = UrlParams{query}
		consts            = map[string]string{}
		page, skip, count uint64
		perPage           = PerPage
		pget              = func(name string) (v string) {
			if v = query.Get(":" + name); v == "" {
				v = r.Header.Get("X-" + utils.NamifyString(name))
			}
			return
		}
		columns, _ = rpt.Schema.Columns()
	)

	if pageS := pget("page"); pageS != "" {
		var err error
		if page, err = strconv.ParseUint(pageS, 10, 64); err != nil {
			http.Error(w, "parse `:page`: "+err.Error(), http.StatusUnprocessableEntity)
			return
		}
	}
	if perPageS := pget("per_page"); perPageS != "" {
		var err error
		if perPage, err = strconv.ParseUint(perPageS, 10, 64); err != nil {
			http.Error(w, "parse `:per_page`: "+err.Error(), http.StatusUnprocessableEntity)
			return
		}
		if perPage > PerPage {
			http.Error(w, "`:per_page` value is greather than 500 records", http.StatusPreconditionFailed)
			return
		}
	}

	if page == 0 {
		page = 1
	}
	if page > 1 {
		skip = (page - 1) * perPage
	}

	consts["OFFSET"] = strconv.FormatUint(skip, 10)
	consts["LIMIT"] = strconv.FormatUint(perPage, 10)

	if count, err = rpt.Count(db, urlp); err != nil {
		http.Error(w, "count records: "+err.Error(), http.StatusPreconditionFailed)
		return
	}

	rows, err := rpt.Find(db, consts, urlp)
	if err != nil {
		http.Error(w, "find failed: "+err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.Header().Set("X-Pages", strconv.FormatUint((count-1)/perPage+1, 10))
	w.Header().Set("X-Per-Page", strconv.FormatUint(perPage, 10))
	w.Header().Set("X-Page", strconv.FormatUint(page, 10))

	var (
		rec        = make(bintb.Recorde, len(columns))
		recScanner = make(bintb.Recorde, len(columns))
	)
	for i, c := range columns {
		rec[i] = c.Tool.Zero()
		recScanner[i] = &rec[i]
	}

	var i uint64 = 0
	next := func() (_ bintb.Recorde, err error) {
		if i == PerPage {
			return nil, nil
		}
		if !rows.Next() {
			return
		}
		if err = rows.Scan(recScanner...); err != nil {
			return
		}
		i++
		return rec, nil
	}

	var (
		spec        = pget("spec") == "true"
		onlyRecords = !(spec || pget("columns") == "true")
	)
	ext := path.Ext(r.URL.Path)
	defer rows.Close()
	switch ext {
	case ".json":
		w.Header().Set("Content-Type", "application/json")
		err = this.DataJson(w, onlyRecords, spec, columns, next)
	case ".csv":
		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
		err = this.DataCsv(w, onlyRecords, spec, columns, next)
	}
	if err != nil && !w.WroteHeader() {
		http.Error(w, "encode results failed: "+err.Error(), http.StatusUnprocessableEntity)
	}
}
