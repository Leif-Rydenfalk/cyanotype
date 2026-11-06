package hcl

import (
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/tychonis/cyanotype/core/process"
	"github.com/tychonis/cyanotype/internal/cerror"
	"github.com/tychonis/cyanotype/internal/digest"
	"github.com/tychonis/cyanotype/internal/stable"
	"github.com/tychonis/cyanotype/model"
)

func (p *Parser) Fuse(sym model.ConcreteSymbol) error {
	var err error
	switch s := sym.(type) {
	case *model.Item:
		s.Digest, err = digest.SHA256FromSymbol(sym)
		if err != nil {
			return err
		}
	case *model.CoItem:
		s.Digest, err = digest.SHA256FromSymbol(sym)
		if err != nil {
			return err
		}
	case *process.Process:
		s.Digest, err = digest.SHA256FromSymbol(sym)
		if err != nil {
			return err
		}
	case *process.CoProcess:
		s.Digest, err = digest.SHA256FromSymbol(sym)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported symbol type")
	}
	return p.Symbols.RegisterConcreteSymbol(sym)
}

func (p *Parser) ProcessSymbol(s *UnprocessedSymbol) (sym model.ConcreteSymbol, err error) {
	// Already processed before.
	if s.qualifier != "" {
		return p.Symbols.FindConcreteSymbol(s.qualifier)
	}

	// New symbol.
	if s.Block == nil || s.Context == nil {
		return nil, errors.New("illegal nil symbol")
	}
	switch s.Block.Type {
	case "item":
		sym, err = p.processItemBlock(s.Context, s.Block)
	case "coitem":
		sym, err = p.processCoItemBlock(s.Context, s.Block)
	case "process":
		sym, err = p.processProcessBlock(s.Context, s.Block)
	case "coprocess":
		sym, err = p.processCoProcessBlock(s.Context, s.Block)
	case "contract":
		sym, err = p.processContractBlock(s.Context, s.Block)
	default:
		return nil, cerror.ErrorWithRange(errors.New("unknown block type"), s.Block.Range())
	}
	if err == nil {
		s.qualifier = sym.GetQualifier()
	} else {
		err = cerror.ErrorWithRange(err, s.Block.Range())
	}
	return
}

func refToQualifier(ctx *ParserContext, ref []string) string {
	if ctx.CurrentModule() == "." {
		return "." + strings.Join(ref, ".")
	} else {
		return strings.Join(append([]string{ctx.CurrentModule()}, ref...), ".")
	}
}

func (p *Parser) resolveContractsLinesAttr(ctx *ParserContext, attr *hcl.Attribute) ([]model.ContractID, error) {
	refs, err := p.readContractLine(ctx, attr.Expr)
	if err != nil {
		return nil, err
	}
	return p.resolveContractsID(ctx, refs)
}

var RESERVED = map[string]struct{}{
	"part_number": {},
	"source":      {},
	"from":        {},
	"impl":        {},
}

func (p *Parser) getDetails(ctx *ParserContext, attrs hcl.Attributes) (stable.Map, error) {
	keys := make([]string, 0)
	for key := range attrs {
		_, ok := RESERVED[key]
		if !ok {
			keys = append(keys, key)
		}
	}
	ret := make(stable.Map)
	for i := 0; i < len(keys); i++ {
		key := keys[i]
		val, err := getString(attrs, key)
		if err != nil {
			return ret, err
		}
		ret[key] = val
	}
	return ret, nil
}

func (p *Parser) processItemBlock(ctx *ParserContext, block *hclsyntax.Block) (*model.Item, error) {
	name := block.Labels[0]
	attrs, err := extractAttributes(block.Body)
	if err != nil {
		return nil, err
	}

	item := &model.Item{}
	item.Type = "item"
	item.Qualifier = ctx.NameToQualifier(name)
	item.Content = &model.ItemContent{
		Name: name,
	}

	item.Content.Details, err = p.getDetails(ctx, attrs)
	if err != nil {
		return item, err
	}

	canonicalNode, err := p.buildCanonicalNode(ctx, item)
	if err != nil {
		return nil, err
	}

	err = p.AssignSubBlocks(ctx, canonicalNode, block)
	if err != nil {
		return nil, err
	}

	err = p.ProcessCanonicalNode(ctx, canonicalNode)
	if err != nil {
		return nil, err
	}

	return item, err
}

func (p *Parser) processCoItemBlock(ctx *ParserContext, block *hclsyntax.Block) (*model.CoItem, error) {
	name := block.Labels[0]
	attrs, err := extractAttributes(block.Body)
	if err != nil {
		return nil, err
	}

	coItem := &model.CoItem{}
	coItem.Type = "coitem"
	coItem.Qualifier = ctx.NameToQualifier(name)
	coItem.Content = &model.ItemContent{
		Name: name,
	}

	reqAttr, ok := attrs["req"]
	if ok {
		coItem.Require, _ = p.resolveContractsLinesAttr(ctx, reqAttr)
	}

	coItem.Content.Details, err = p.getDetails(ctx, attrs)
	if err != nil {
		return coItem, err
	}

	subContent, err := p.ProcessSubBlocks(ctx, block)
	if err != nil {
		return coItem, err
	}
	coItem.Content.Artifacts = subContent.Artifacts

	err = p.Fuse(coItem)
	return coItem, err
}

func (p *Parser) processProcessBlock(ctx *ParserContext, block *hclsyntax.Block) (ret *process.Process, err error) {
	name := block.Labels[0]
	ret = &process.Process{}
	ret.Type = "process"
	ret.Qualifier = ctx.NameToQualifier(name)
	_, diags := block.Body.JustAttributes()
	if diags.HasErrors() {
		return nil, diags
	}
	err = p.Fuse(ret)
	return
}

func (p *Parser) processCoProcessBlock(ctx *ParserContext, block *hclsyntax.Block) (ret *process.CoProcess, err error) {
	name := block.Labels[0]
	ret = &process.CoProcess{}
	ret.Type = "coprocess"
	ret.Qualifier = ctx.NameToQualifier(name)
	_, diags := block.Body.JustAttributes()
	if diags.HasErrors() {
		return nil, diags
	}
	err = p.Fuse(ret)
	return
}
