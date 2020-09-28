package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Knetic/govaluate"
	"unapu.com/sql-rpt/rpt"
)

func help() {
	fmt.Fprintf(os.Stderr, `%s --help | REPORT_FILE [--bindv=BINDV] [--param_key=param_value]...

REPORT_FILE: the report file path or - (from stdin)

BINDV:
  Your database bind variable expression [1]. Default value is '"$"+i'.
  Example: '"$"+i' -> produces '$1', '$2' etc

  [1] https://github.com/Knetic/govaluate.
`, filepath.Base(os.Args[0]))
}

func main() {
	var sqlb = &rpt.SqlBuilder{
		Params: map[string][]string{},
		BindVar: func(i int) string {
			return fmt.Sprintf("$%d", i)
		},
	}
	if len(os.Args) < 2 {
		help()
		os.Exit(-1)
	}
	sqlb.Path = os.Args[1]

	for _, arg := range os.Args[2:] {
		arg = strings.TrimPrefix(arg, "--")
		if arg == "help" {
			help()
			os.Exit(0)
		}
		parts := strings.SplitN(arg, "=", 2)
		switch len(parts) {
		case 2:
			switch parts[0] {
			case "bindv":
				expr, err := govaluate.NewEvaluableExpression(parts[1])
				if err != nil {
					fmt.Fprintf(os.Stderr, "parse bind var expresison: %s\n", err.Error())
					os.Exit(-1)
				}
				func(expr *govaluate.EvaluableExpression) {
					sqlb.BindVar = func(i int) string {
						result, err := expr.Evaluate(map[string]interface{}{
							"i": i,
						})
						if err != nil {
							fmt.Fprintf(os.Stderr, "bindvar evaluation: %s\n", err.Error())
							os.Exit(-1)
						}
						return fmt.Sprint(result)
					}
				}(expr)
			default:
				if _, ok := sqlb.Params[parts[0]]; !ok {
					sqlb.Params[parts[0]] = []string{parts[1]}
				} else {
					sqlb.Params[parts[0]] = append(sqlb.Params[parts[0]], parts[1])
				}
			}
		}
	}

	if err := sqlb.Build(); err != nil {
		fmt.Fprintf(os.Stderr, "build: %s\n", err.Error())
		os.Exit(-1)
	}

	query, args, err := sqlb.Counter()
	if err != nil {
		fmt.Fprintf(os.Stderr, "build counter: %s\n", err.Error())
		os.Exit(-1)
	}
	fmt.Println("===== COUNTER =====")
	fmt.Println(query)
	if len(args) > 0 {
		fmt.Println("\nParameters:")
	}
	for i, arg := range args {
		fmt.Fprintf(os.Stdout, "  %s: %s", sqlb.BindVar(i+1), arg)
	}

	if query, args, err = sqlb.Finder(); err != nil {
		fmt.Fprintf(os.Stderr, "build counter: %s\n", err.Error())
		os.Exit(-1)
	}
	fmt.Println("===== FINDER =====")
	fmt.Println(query)
	if len(args) > 0 {
		fmt.Println("\nParameters:")
	}
	for i, arg := range args {
		fmt.Fprintf(os.Stdout, "  %s: %s", sqlb.BindVar(i+1), arg)
	}
}
