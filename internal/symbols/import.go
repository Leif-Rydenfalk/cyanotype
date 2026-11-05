package symbols

import (
	"errors"

	"github.com/tychonis/cyanotype/model"
)

type Import struct {
	Identifier string
	Symbols    *SymbolTable
}

func (i *Import) Resolve(path []string) (model.Resolver, error) {
	m, ok := i.Symbols.Modules[i.Identifier].Symbols[path[0]]
	if !ok {
		return nil, errors.New("symbol not existed")
	}
	return m.Resolve(path[1:])
}
