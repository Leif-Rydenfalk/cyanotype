package hcl

import (
	"fmt"

	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func (p *Parser) processComponentBlock(ctx *ParserContext, block *hclsyntax.Block) (*UnresolvedBOMLine, error) {
	name := block.Labels[0]

	ret := &UnresolvedBOMLine{
		Name: name,
		Qty:  1,
	}

	attrs, diags := block.Body.JustAttributes()
	if diags.HasErrors() {
		return nil, diags
	}

	if attr, ok := attrs["ref"]; ok {
		ref, err := exprToRef(ctx, attr.Expr)
		if err != nil {
			return nil, err
		}
		ret.Ref = ref
	} else {
		return nil, fmt.Errorf("component %q requires ref", ret.Name)
	}

	if attr, ok := attrs["qty"]; ok {
		val, diags := attr.Expr.Value(nil)
		if diags.HasErrors() {
			return nil, diags
		}
		ret.Qty, _ = val.AsBigFloat().Float64()
	}

	if attr, ok := attrs["placement"]; ok {
		val, diags := attr.Expr.Value(nil)
		if diags.HasErrors() {
			return nil, diags
		}

		slice := val.AsValueSlice()
		if len(slice) != 7 {
			return nil, fmt.Errorf("placement must contain 7 numbers")
		}

		ret.HasPlacement = true
		for i, num := range slice[:4] {
			ret.Placement.Rotation[i], _ = num.AsBigFloat().Float64()
		}
		for i, num := range slice[4:] {
			ret.Placement.Position[i], _ = num.AsBigFloat().Float64()
		}
	}

	return ret, nil
}
