package hcl

import (
	"errors"
	"log/slog"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/tychonis/cyanotype/core/process"
	"github.com/tychonis/cyanotype/core/qualifier"
	"github.com/tychonis/cyanotype/model"
)

type SymbolContext[T model.ConcreteSymbol] struct {
	Symbol T
	Blocks []*hclsyntax.Block
}

type CanonicalNode struct {
	ExplicitBlock *hclsyntax.Block
	CoItem        *SymbolContext[*model.CoItem]
	CoProcess     *SymbolContext[*process.CoProcess]
	Item          *SymbolContext[*model.Item]
	Process       *SymbolContext[*process.Process]
}

func (p *Parser) buildCanonicalNode(ctx *ParserContext, item *model.Item) (*CanonicalNode, error) {
	slog.Debug("building canonical node", "module", ctx.CurrentModule(), "item", item.Qualifier)

	canonicalNode := &CanonicalNode{}

	canonicalNode.Item = &SymbolContext[*model.Item]{
		Symbol: item,
		Blocks: make([]*hclsyntax.Block, 0),
	}

	co := &model.CoItem{}
	co.Type = "coitem"
	co.Qualifier = qualifier.ImplicitCoItem(item)
	co.Content = &model.ItemContent{
		Name: "Implicit requirements for " + item.Content.Name,
	}
	canonicalNode.CoItem = &SymbolContext[*model.CoItem]{
		Symbol: co,
		Blocks: make([]*hclsyntax.Block, 0),
	}

	cp := &process.CoProcess{}
	cp.Type = "coprocess"
	cp.Qualifier = qualifier.ImplicitCoProcess(item)
	canonicalNode.CoProcess = &SymbolContext[*process.CoProcess]{
		Symbol: cp,
		Blocks: make([]*hclsyntax.Block, 0),
	}

	pr := &process.Process{}
	pr.Type = "process"
	pr.Qualifier = qualifier.ImplicitProcess(item)
	canonicalNode.Process = &SymbolContext[*process.Process]{
		Symbol: pr,
		Blocks: make([]*hclsyntax.Block, 0),
	}

	return canonicalNode, nil
}

// AssignSubBlocks assigns the sub-blocks of a given HCL block to the appropriate sections of the canonical node.
// Current assumption is that this is only applied for item blocks as syntax sugar.
func (p *Parser) AssignSubBlocks(ctx *ParserContext, node *CanonicalNode, block *hclsyntax.Block) error {
	node.ExplicitBlock = block
	for _, child := range block.Body.Blocks {
		switch child.Type {
		case "artifact":
			node.Item.Blocks = append(node.Item.Blocks, child)
		case "contract":
			node.Item.Blocks = append(node.Item.Blocks, child)
			node.Process.Blocks = append(node.Process.Blocks, child)
		case "operation":
			node.Process.Blocks = append(node.Process.Blocks, child)
		case "component":
			node.Process.Blocks = append(node.Process.Blocks, child)
		default:
			slog.Warn("Unrecognized sub block, skipping", slog.String("type", child.Type))
			continue
		}
	}
	return nil
}

func (p *Parser) processItemArtifactBlock(ctx *ParserContext, item *model.Item, block *hclsyntax.Block) error {
	artifact, err := p.processArtifactBlock(ctx, block)
	if err != nil {
		return err
	}
	if item.Content == nil {
		item.Content = &model.ItemContent{}
	}
	if item.Content.Artifacts == nil {
		item.Content.Artifacts = make([]*model.Artifact, 0)
	}
	item.Content.Artifacts = append(item.Content.Artifacts, artifact)
	return nil
}

func (p *Parser) processItemSubBlocks(ctx *ParserContext, symCtx *SymbolContext[*model.Item]) error {
	for _, block := range symCtx.Blocks {
		switch block.Type {
		case "artifact":
			err := p.processItemArtifactBlock(ctx, symCtx.Symbol, block)
			if err != nil {
				return err
			}
		case "contract":
			continue
		default:
			slog.Warn("Unsupported block for item, skipping", slog.String("type", block.Type))
			continue
		}
	}
	return nil
}

func (p *Parser) processCoItemSubBlocks(ctx *ParserContext, symCtx *SymbolContext[*model.CoItem]) error {
	for _, block := range symCtx.Blocks {
		switch block.Type {
		default:
			slog.Warn("Unsupported block for co-item, skipping", slog.String("type", block.Type))
			continue
		}
	}
	return nil
}

func (p *Parser) processCoProcessSubBlocks(ctx *ParserContext, symCtx *SymbolContext[*process.CoProcess]) error {
	for _, block := range symCtx.Blocks {
		switch block.Type {
		default:
			slog.Warn("Unsupported block for co-process, skipping", slog.String("type", block.Type))
			continue
		}
	}
	return nil
}

func (p *Parser) processCompanionProcess(ctx *ParserContext, node *CanonicalNode) error {
	components, err := p.extractUnresolvedBOMLines(ctx, node.ExplicitBlock)
	if err != nil {
		return err
	}
	for _, block := range node.Process.Blocks {
		switch block.Type {
		case "contract":
			continue
		case "operation":
			continue
		case "component":
			continue
		default:
			slog.Warn("Unsupported block for process, skipping", slog.String("type", block.Type))
			continue
		}
	}
	pc, err := p.resolvingBOMLines(ctx, components)
	if err != nil {
		return err
	}
	switch content := pc.(type) {
	case *process.Abstract:
		content.Name = "Implicit process for " + node.Item.Symbol.Content.Name
		content.Output = []*model.BOMLine{
			{
				Item: node.Item.Symbol.Digest,
				Qty:  1,
			},
		}
	case *process.Drawing:
		content.Name = "Implicit drawing for " + node.Item.Symbol.Content.Name
		content.Output = []*model.BOMLine{
			{
				Item: node.Item.Symbol.Digest,
				Qty:  1,
			},
		}
	default:
		return errors.New("process content type not recognized")
	}

	node.Process.Symbol.Content = pc
	return nil
}

// extractUnresolvedBOMLines extracts the unresolved BOM lines from the canonical node using the "from" attribute if present.
// If the "from" attribute is not present, it returns an abstract process content.
// TODO: Deprecate this function in favor of sub block definition "component"
func (p *Parser) extractUnresolvedBOMLines(ctx *ParserContext, explicitBlock *hclsyntax.Block) ([]*UnresolvedBOMLine, error) {
	attrs, err := extractAttributes(explicitBlock.Body)
	if err != nil {
		return nil, err
	}
	var from []*UnresolvedBOMLine
	fromAttr, ok := attrs["from"]
	if ok {
		from = parseBOMLinesAttr(ctx, fromAttr)
	}
	return from, nil
}

func (p *Parser) ProcessCanonicalNode(ctx *ParserContext, node *CanonicalNode) error {
	err := p.processItemSubBlocks(ctx, node.Item)
	if err != nil {
		return err
	}

	err = p.Fuse(node.Item.Symbol)
	if err != nil {
		return err
	}

	err = p.processCoItemSubBlocks(ctx, node.CoItem)
	if err != nil {
		return err
	}

	err = p.Fuse(node.CoItem.Symbol)
	if err != nil {
		return err
	}

	err = p.processCoProcessSubBlocks(ctx, node.CoProcess)
	if err != nil {
		return err
	}

	if node.CoProcess.Symbol.Content == nil {
		content := &process.Abstract{
			Name: "Implicit co-process for " + node.Item.Symbol.Content.Name,
			Input: []*model.BOMLine{
				{
					Item: node.Item.Symbol.Digest,
					Qty:  1,
				},
			},
			Output: []*model.BOMLine{
				{
					Item: node.CoItem.Symbol.Digest,
					Qty:  1,
				},
			},
		}
		node.CoProcess.Symbol.Content = content
	}

	err = p.Fuse(node.CoProcess.Symbol)
	if err != nil {
		return err
	}

	err = p.processCompanionProcess(ctx, node)
	if err != nil {
		return err
	}

	err = p.Fuse(node.Process.Symbol)
	if err != nil {
		return err
	}

	return nil
}
