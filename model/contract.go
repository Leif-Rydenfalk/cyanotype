package model

import (
	"github.com/tychonis/cyanotype/internal/stable"
)

type ContractID = Digest

type Contract struct {
	Type      string     `json:"type" yaml:"type"`
	Category  string     `json:"category" yaml:"category"`
	Qualifier string     `json:"qualifier" yaml:"qualifier"`
	Name      string     `json:"name" yaml:"name"`
	Params    stable.Map `json:"params" yaml:"params"`

	Digest ContractID `json:"-" yaml:"-"`
}

func (c *Contract) Fulfill(c2 *Contract) bool {
	return false
}

func (c *Contract) GetQualifier() string {
	return c.Qualifier
}

func (c *Contract) GetDigest() string {
	return c.Digest
}

func (c *Contract) GetType() string {
	return c.Type
}
