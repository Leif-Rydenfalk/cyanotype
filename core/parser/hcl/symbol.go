package hcl

import (
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/tychonis/cyanotype/model"
)

type UnprocessedSymbol struct {
	Context *ParserContext
	Block   *hclsyntax.Block

	qualifier string
}

func (us *UnprocessedSymbol) Resolve(path []string) (model.Resolver, error) {
	return us, nil
}
