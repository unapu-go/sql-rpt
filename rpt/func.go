package rpt

import (
	"bytes"
	"strings"

	"github.com/pkg/errors"
)


func (this *ArgsBuilder) NewFunc(def, body string) {
	var (
		name string
		argsNames []string
	)

	pos := strings.IndexByte(def, '(')
	if pos > 0 {
		name = def[0:pos]
		argsNames = strings.Split(def[pos+1:len(def)-1], ",")
	} else {
		name = def
	}
	if len(argsNames) == 0 {
		this.efuncs[name] = func(arguments ...interface{}) (interface{}, error) {
			var w bytes.Buffer
			if err := this.ExecuteTemplate(&w, body); err != nil {
				return nil, errors.Wrap(err, def)
			}
			return Raw{w.String()}, nil
		}
		return
	}
	this.efuncs[name] = func(arguments ...interface{}) (interface{}, error) {
		var args = map[string]interface{}{}
		for i, name := range argsNames {
			args[name] = arguments[i]
		}
		this.funcArgs = append(this.funcArgs, args)
		defer func() {
			this.funcArgs = this.funcArgs[0:len(this.funcArgs)-1]
		}()
		var w bytes.Buffer
		if err := this.ExecuteTemplate(&w, body); err != nil {
			return nil, errors.Wrap(err, def)
		}
		return Raw{w.String()}, nil
	}
}
