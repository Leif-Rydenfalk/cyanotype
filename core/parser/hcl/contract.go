package hcl

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/tychonis/cyanotype/internal/digest"
	"github.com/tychonis/cyanotype/model"
)

func (p *Parser) getParams(ctx *ParserContext, cat string, attrs hcl.Attributes) (map[string]any, error) {
	params := make(map[string]any)
	if cat == "point" {
	}
	return params, nil
}

func (p *Parser) processContractBlock(ctx *ParserContext, block *hclsyntax.Block) (*model.Contract, error) {
	cat := block.Labels[0]
	name := block.Labels[1]

	attrs, diags := block.Body.JustAttributes()
	if diags.HasErrors() {
		return nil, diags
	}
	params, err := p.getParams(ctx, cat, attrs)
	if err != nil {
		return nil, err
	}
	contract := &model.Contract{
		Type:      "contract",
		Category:  cat,
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
