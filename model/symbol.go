package model

type Qualifier = string

type Resolver interface {
	Resolve(path []string) (Resolver, error)
}

type ConcreteSymbol interface {
	GetType() string
	GetQualifier() Qualifier
	GetDigest() Digest
}

type BOMLine struct {
	Name string  `json:"name" yaml:"name"`
	Item ItemID  `json:"item" yaml:"item"`
	Qty  float64 `json:"qty" yaml:"qty"`
}
