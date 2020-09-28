package rpt

import (
	"fmt"
	"io"

	"github.com/Knetic/govaluate"
	"github.com/pkg/errors"
	"github.com/valyala/fasttemplate"
)

type NoParamError struct {
	Name string
}

func (n NoParamError) Error() string {
	return fmt.Sprintf("no parameter %q", n.Name)
}

type Raw struct {
	Value interface{}
}

type ArgsBuilder struct {
	bindVar func(i int) string
	Result  []interface{}
	resultM map[string]string
	rawArgs []map[string]string
	params  []Paramer
	efuncs map[string]govaluate.ExpressionFunction
	funcArgs []map[string]interface{}
	Declare map[string]interface{}
}

func NewArgsBuilder(bindVar func(i int) string, funcs map[string]string, rawArgs []map[string]string, params []Paramer) *ArgsBuilder {
	ab := &ArgsBuilder{bindVar: bindVar, rawArgs: rawArgs, params: params, resultM: map[string]string{}}
	ab.efuncs = map[string]govaluate.ExpressionFunction {
		"sql": func(args ...interface{}) (interface{}, error) {
			return Raw{args[0]}, nil
		},
		"get": func(args ...interface{}) (interface{}, error) {
			if len(args) == 0 {
				return nil, nil
			}
			switch t := args[0].(type) {
			case string:
				if v, err := ab.Get(args[0].(string)); err != nil {
					if npe, ok := err.(NoParamError); ok && npe.Name == t {
						return nil, nil
					}
					return nil, errors.Wrapf(err, "get(%q)", t)
				} else {
					return v, nil
				}
			default:
				return nil, fmt.Errorf("bad get(<string>) argument: %T{%v}", args[0], args[0])
			}
		},
		"arg": func(arguments ...interface{}) (interface{}, error) {
			if len(arguments) == 1 {
				switch t := arguments[0].(type) {
				case string:
					if len(ab.funcArgs) > 0 {
						if arg, ok := ab.funcArgs[len(ab.funcArgs)-1][t]; ok {
							return Raw{fmt.Sprint(arg)}, nil
						}
						return nil, fmt.Errorf("argument %q not found}", t)
					}
				}
				return nil, fmt.Errorf("bad arg(<string>) argument: %T{%v}", arguments[0], arguments[0])
			}
			return nil, fmt.Errorf("bad arg(<string>) invokation: %s", arguments)
		},
		"NULL": func(args ...interface{}) (interface{}, error) {
			return Raw{"NULL"}, nil
		},
	}
	if funcs != nil {
		for def, body := range funcs {
			ab.NewFunc(def, body)
		}
	}
	return ab
}

func (this *ArgsBuilder) Eval(expr string) (s interface{}, _ error) {
	if v, ok := this.resultM[expr]; ok {
		return v, nil
	}

	expression, err := govaluate.NewEvaluableExpressionWithFunctions(expr, this.efuncs)
	if err != nil {
		return "", err
	}
	if result, err := expression.Eval(this); err != nil {
		return nil, err
	} else {
		if this.Result == nil {
			return result, nil
		}
		if raw, ok := result.(Raw); ok {
			return fmt.Sprint(raw.Value), nil
		}
		this.Result = append(this.Result, result)
		repl := this.bindVar(len(this.Result))
		this.resultM[expr] = repl
		return repl, nil
	}
}

func (this ArgsBuilder) Get(name string) (i interface{}, err error) {
	if v := this.get(name); v != nil {
		return v, nil
	}
	return nil, NoParamError{name}
}

func (this ArgsBuilder) get(name string) (v interface{}) {
	var ok bool

	if name[0] == '_' {
		name = name[1:]
		var vs []string
		for _, params := range this.params {
			if vs, ok = params.Values(name); ok {
				return Raw{vs}
			}
		}
	} else {
		var s string
		for _, params := range this.params {
			if s, ok = params.Value(name); ok {
				return s
			}
		}
	}
	if value, ok := this.Declare[name]; ok {
		return value
	}
	for _, rawArgs := range this.rawArgs {
		if c, ok := rawArgs[name]; ok {
			return Raw{c}
		}
	}
	return
}

func (this *ArgsBuilder) ExecuteTemplate(w io.Writer, template string) (err error) {
	_, err = fasttemplate.New(template, "${", "}").ExecuteFunc(w, func(w io.Writer, tag string) (int, error) {
		if s, err := this.Eval(tag); err != nil {
			return 0, err
		} else {
			return w.Write([]byte(fmt.Sprint(s)))
		}
	})
	return
}