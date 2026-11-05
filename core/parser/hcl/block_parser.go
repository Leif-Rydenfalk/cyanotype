package hcl

import (
	"errors"
	"log/slog"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/tychonis/cyanotype/core/process"
	"github.com/tychonis/cyanotype/internal/cerror"
	"github.com/tychonis/cyanotype/internal/digest"
	"github.com/tychonis/cyanotype/internal/stable"
	"github.com/tychonis/cyanotype/model"
)

func (p *Parser) ParseSymbol(s *UnprocessedSymbol) (sym model.ConcreteSymbol, err error) {
	// Already parsed before.
	if s.qualifier != "" {
		return p.Symbols.FindConcreteSymbol(s.qualifier)
	}

	// New symbol.
	if s.Block == nil || s.Context == nil {
		return nil, errors.New("illegal nil symbol")
	}
	switch s.Block.Type {
	case "item":
		sym, err = p.parseItemBlock(s.Context, s.Block)
	case "coitem":
		sym, err = p.parseCoItemBlock(s.Context, s.Block)
	case "process":
		sym, err = p.parseProcessBlock(s.Context, s.Block)
	case "coprocess":
		sym, err = p.parseCoProcessBlock(s.Context, s.Block)
	case "contract":
		sym, err = p.parseContractBlock(s.Context, s.Block)
	default:
		return nil, cerror.ErrorWithRange(errors.New("unknown block type"), s.Block.Range())
	}
	if err == nil {
		// TODO: remove side effects here.
		s.qualifier = sym.GetQualifier()
		slog.Debug("saving symbol",
			"qualifier", sym.GetQualifier(), "digest", sym.GetDigest())
		p.Symbols.RegisterConcreteSymbol(sym)
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

func (p *Parser) parseItemBlock(ctx *ParserContext, block *hclsyntax.Block) (*model.Item, error) {
	name := block.Labels[0]
	attrs, err := extractAttributes(block.Body)
	if err != nil {
		return nil, err
	}
	pn, _ := getString(attrs, "part_number")
	src, _ := getString(attrs, "source")

	var pc process.ProcessContent
	fromAttr, ok := attrs["from"]
	if ok {
		from := parseBOMLinesAttr(ctx, fromAttr)
		pc, err = p.processKeywordFROM(ctx, from)
		if err != nil {
			return nil, err
		}
	} else {
		pc = &process.Abstract{}
	}

	item := &model.Item{}
	item.Type = "item"
	item.Qualifier = ctx.NameToQualifier(name)
	item.Content = &model.ItemContent{
		Name:       name,
		Source:     src,
		PartNumber: pn,
	}

	implAttr, ok := attrs["impl"]
	if ok {
		item.Implement, _ = p.resolveContractsLinesAttr(ctx, implAttr)
	}

	item.Content.Details, err = p.getDetails(ctx, attrs)
	if err != nil {
		return item, err
	}

	subContent, err := p.ParseSubBlocks(ctx, block)
	if err != nil {
		return item, err
	}
	item.Content.Artifacts = subContent.Artifacts

	// Digest need to be computed before building companion processes,
	// because the companion processes will reference the item digest.
	item.Digest, err = digest.SHA256FromSymbol(item)
	if err != nil {
		return item, err
	}

	companion, err := p.buildCompanionForItem(ctx, item, pc)
	if err != nil {
		return nil, err
	}
	for _, s := range companion {
		err := p.Symbols.RegisterConcreteSymbol(s)
		if err != nil {
			return nil, err
		}
	}
	return item, err
}

func (p *Parser) parseCoItemBlock(ctx *ParserContext, block *hclsyntax.Block) (*model.CoItem, error) {
	name := block.Labels[0]
	attrs, err := extractAttributes(block.Body)
	if err != nil {
		return nil, err
	}
	pn, _ := getString(attrs, "part_number")
	src, _ := getString(attrs, "source")

	coItem := &model.CoItem{}
	coItem.Type = "coitem"
	coItem.Qualifier = ctx.NameToQualifier(name)
	coItem.Content = &model.ItemContent{
		Name:       name,
		Source:     src,
		PartNumber: pn,
	}

	reqAttr, ok := attrs["req"]
	if ok {
		coItem.Require, _ = p.resolveContractsLinesAttr(ctx, reqAttr)
	}

	coItem.Content.Details, err = p.getDetails(ctx, attrs)
	if err != nil {
		return coItem, err
	}

	subContent, err := p.ParseSubBlocks(ctx, block)
	if err != nil {
		return coItem, err
	}
	coItem.Content.Artifacts = subContent.Artifacts

	coItem.Digest, err = digest.SHA256FromSymbol(coItem)
	return coItem, err
}

func (p *Parser) parseProcessBlock(ctx *ParserContext, block *hclsyntax.Block) (ret *process.Process, err error) {
	name := block.Labels[0]
	ret = &process.Process{}
	ret.Type = "process"
	ret.Qualifier = ctx.NameToQualifier(name)
	_, diags := block.Body.JustAttributes()
	if diags.HasErrors() {
		return nil, diags
	}
	ret.Digest, err = digest.SHA256FromSymbol(ret)
	return
}

func (p *Parser) parseCoProcessBlock(ctx *ParserContext, block *hclsyntax.Block) (ret *process.CoProcess, err error) {
	name := block.Labels[0]
	ret = &process.CoProcess{}
	ret.Type = "coprocess"
	ret.Qualifier = ctx.NameToQualifier(name)
	_, diags := block.Body.JustAttributes()
	if diags.HasErrors() {
		return nil, diags
	}
	ret.Digest, err = digest.SHA256FromSymbol(ret)
	return
}
