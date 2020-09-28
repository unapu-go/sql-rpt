package rpt

import (
	"github.com/pkg/errors"

	"github.com/moisespsena-go/bintb"
)

type Field struct {
	Name  string
	Title string
	Type  bintb.ColumnType
}

type Schema struct {
	Fields  []Field
	columns []*bintb.Column
}

func (this *Schema) Columns() (columns []*bintb.Column, err error) {
	if len(this.columns) > 0 {
		return this.columns, nil
	}
	columns = make([]*bintb.Column, len(this.Fields))
	for i, f := range this.Fields {
		if f.Type == "" {
			f.Type = bintb.CtString
		}
		if bintb.ColumnTypeTool.Get(f.Type) == nil {
			return nil, errors.Errorf("field %d[%q] unrecognized type %q", i, f.Name, f.Type)
		}
		columns[i] = bintb.NewColumn(f.Name, f.Type, bintb.AllowBlank)
	}
	this.columns = columns
	return
}
