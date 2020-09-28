package rpt

import (
	"bytes"

	"github.com/pkg/errors"
)

type Query struct {
	Query   string
	Args    map[string][]string
	RawArgs map[string]string `yaml:"raw_args"`
	Declare []map[string]string
	Funcs   map[string]string
}

func (this Query) Build(bindVar func(i int) string, consts map[string]string, params Paramer) (query string, args []interface{}, err error) {
	//ql := strings.ToLower(this.Query)
	/*for _, term := range []string{"update", "drop", "delete", "alter", "insert", "execute", "create"} {
		if strings.Contains(ql, term) {
			err = errors.New("unsafe query not allowed")
			return
		}
	}*/
	var w bytes.Buffer
	var declare = map[string]interface{}{}
	argsb := NewArgsBuilder(bindVar, this.Funcs, []map[string]string{this.RawArgs, consts}, []Paramer{params, UrlParams{this.Args}})
	argsb.Declare = declare
	for _, m := range this.Declare {
		for key, value := range m {
			var s interface{}
			if s, err = argsb.Eval(value); err != nil {
				err = errors.Errorf("declare %q: %s", key, err.Error())
				return
			}
			declare[key] = s
		}
	}
	argsb.Result = []interface{}{}

	if err = argsb.ExecuteTemplate(&w, this.Query); err == nil {
		return w.String(), argsb.Result, nil
	}
	return
}