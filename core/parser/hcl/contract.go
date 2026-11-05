package hcl

import (
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/tychonis/cyanotype/internal/digest"
	"github.com/tychonis/cyanotype/model"
)

func (p *Parser) parseContractBlock(ctx *ParserContext, block *hclsyntax.Block) (*model.Contract, error) {
	name := block.Labels[0]

	attrs, diags := block.Body.JustAttributes()
	if diags.HasErrors() {
		return nil, diags
	}
	params := make(map[string]any)
	for attr := range attrs {
		// TODO: support more types?
		param, _ := getString(attrs, attr)
		params[attr] = param
	}
	contract := &model.Contract{
		Type:      "contract",
		Name:      name,
		Qualifier: ctx.NameToQualifier(name),
		Params:    params,
	}
	digest, err := digest.SHA256FromSymbol(contract)
	if err != nil {
		return contract, err
	}
	contract.Digest = digest
	return contract, nil
}
