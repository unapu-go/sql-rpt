package rpt

import (
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

func GetReport(name string, src ...string) (rpt *Report, err error) {
	for _, src := range src {
		for _, ext := range []string{".yml", ".yaml"} {
			src = filepath.Join(src, "sql-reports", name+ext)
			var f *os.File
			if f, err = os.Open(src); err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, errors.Wrapf(err, "report %q", name)
			} else {
				defer f.Close()
				if rpt, err = Load(f); err != nil {
					err = errors.Wrapf(err, "report %q", name)
				}
				return
			}
		}
	}
	return nil, os.ErrNotExist
}

func Load(r io.Reader) (rpt *Report, err error) {
	rpt = &Report{}
	if err = yaml.NewDecoder(r).Decode(rpt); err != nil {
		return
	}
	if len(rpt.Schema.Fields) == 0 {
		return nil, errors.New("no fields")
	}
	rpt.Counter.Declare = append(rpt.Declare, rpt.Counter.Declare...)
	rpt.Finder.Declare = append(rpt.Declare, rpt.Finder.Declare...)
	if rpt.RawArgs != nil {
		if rpt.Counter.RawArgs == nil {
			rpt.Counter.RawArgs = map[string]string{}
		}
		if rpt.Finder.RawArgs == nil {
			rpt.Finder.RawArgs = map[string]string{}
		}
		for key, value := range rpt.RawArgs {
			if _, ok := rpt.Counter.RawArgs[key]; !ok {
				rpt.Counter.RawArgs[key] = value
			}
			if _, ok := rpt.Finder.RawArgs[key]; !ok {
				rpt.Finder.RawArgs[key] = value
			}
		}
	}
	if rpt.Funcs != nil {
		if rpt.Counter.Funcs == nil {
			rpt.Counter.Funcs = map[string]string{}
		}
		if rpt.Finder.Funcs == nil {
			rpt.Finder.Funcs = map[string]string{}
		}
		for key, value := range rpt.Funcs {
			if _, ok := rpt.Counter.Funcs[key]; !ok {
				rpt.Counter.Funcs[key] = value
			}
			if _, ok := rpt.Finder.Funcs[key]; !ok {
				rpt.Finder.Funcs[key] = value
			}
		}
	}
	return rpt, nil
}
