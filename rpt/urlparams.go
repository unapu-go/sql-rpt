package rpt

type UrlParams struct {
	UrlValues map[string][]string
}

func (this UrlParams) Values(name string) (v []string, ok bool) {
	v, ok = this.UrlValues[name]
	if ok && len(v) == 0 {
		ok = false
	}
	return
}

func (this UrlParams) Value(name string) (string, bool) {
	v, ok := this.Values(name)
	if !ok {
		return "", false
	}
	return v[0], true
}
