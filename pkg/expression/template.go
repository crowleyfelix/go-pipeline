package expression

import (
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

var templ *template.Template

func init() {
	templ = template.New("").
		Funcs(sprig.FuncMap())
}

func RegisterFuncs(funcs template.FuncMap) {
	templ = templ.Funcs(funcs)
}
