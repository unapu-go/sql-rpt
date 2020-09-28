package rpt

type DBConfig struct {
	BindVar func(i int) string
	DB DB
}