package symbols

import (
	"fmt"

	"github.com/tychonis/cyanotype/model"
)

type SymbolTable struct {
	Modules         map[string]*ModuleScope
	ConcreteSymbols map[model.Qualifier]model.ConcreteSymbol
}

type ModuleScope struct {
	Symbols map[string]model.Symbol
}

func NewSymbolTable() *SymbolTable {
	return &SymbolTable{
		Modules:         make(map[string]*ModuleScope),
		ConcreteSymbols: make(map[model.Qualifier]model.ConcreteSymbol),
	}
}

func NewModuleScope() *ModuleScope {
	return &ModuleScope{Symbols: make(map[string]model.Symbol)}
}

func (t *SymbolTable) AddSymbol(module string, name string, symbol model.Symbol) error {
	if t.Modules[module] == nil {
		t.Modules[module] = NewModuleScope()
	}
	_, ok := t.Modules[module].Symbols[name]
	if ok {
		return fmt.Errorf("symbol %s already existed in %s", name, module)
	}
	t.Modules[module].Symbols[name] = symbol
	return nil
}

func (t *SymbolTable) RegisterConcreteSymbol(sym model.ConcreteSymbol) error {
	t.ConcreteSymbols[sym.GetQualifier()] = sym
	return nil
}

var ErrNotFound = fmt.Errorf("symbol not found")

func (t *SymbolTable) FindConcreteSymbol(qualifier model.Qualifier) (model.ConcreteSymbol, error) {
	sym, ok := t.ConcreteSymbols[qualifier]
	if !ok {
		return nil, ErrNotFound
	}
	return sym, nil
}

func (m *ModuleScope) Resolve(ref []string) (model.Symbol, error) {
	if len(ref) <= 0 {
		return m, nil
	}
	sym := ref[0]
	resolver, ok := m.Symbols[sym]
	if !ok {
		return nil, fmt.Errorf("symbol %v not registered", ref)
	}
	return resolver.Resolve(ref[1:])
}
