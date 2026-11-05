package hcl

import (
	"errors"
	"log/slog"

	"github.com/tychonis/cyanotype/core/process"
	"github.com/tychonis/cyanotype/core/qualifier"
	"github.com/tychonis/cyanotype/internal/digest"
	"github.com/tychonis/cyanotype/model"
)

type Companion []model.ConcreteSymbol

func (p *Parser) buildCompanionCoItem(item *model.Item) (*model.CoItem, error) {
	var err error
	co := &model.CoItem{}
	co.Type = "coitem"
	co.Qualifier = qualifier.ImplicitCoItem(item)
	co.Content = &model.ItemContent{
		Name: "Implicit requirements for " + item.Content.Name,
	}
	co.Digest, err = digest.SHA256FromSymbol(co)
	return co, err
}

func (p *Parser) buildCompanionCoProcess(item *model.Item, coItem *model.CoItem) (*process.CoProcess, error) {
	var err error
	content := &process.Abstract{
		Name: "Implicit co-process for " + item.Content.Name,
		Input: []*model.BOMLine{
			{
				Item: item.Digest,
				Qty:  1,
			},
		},
		Output: []*model.BOMLine{
			{
				Item: coItem.Digest,
				Qty:  1,
			},
		},
	}

	cp := &process.CoProcess{}
	cp.Type = "coprocess"
	cp.Qualifier = qualifier.ImplicitCoProcess(item)
	cp.Content = content
	cp.Digest, err = digest.SHA256FromSymbol(cp)
	return cp, err
}

func (p *Parser) buildCompanionProcess(item *model.Item, pc process.ProcessContent) (*process.Process, error) {
	var err error
	switch content := pc.(type) {
	case *process.Abstract:
		content.Name = "Implicit process for " + item.Content.Name
		content.Output = []*model.BOMLine{
			{
				Item: item.Digest,
				Qty:  1,
			},
		}
	case *process.Drawing:
		content.Name = "Implicit drawing for " + item.Content.Name
		content.Output = []*model.BOMLine{
			{
				Item: item.Digest,
				Qty:  1,
			},
		}
	default:
		return nil, errors.New("process content type not recognized")
	}

	process := &process.Process{}
	process.Type = "process"
	process.Qualifier = qualifier.ImplicitProcess(item)
	process.Content = pc
	process.Digest, err = digest.SHA256FromSymbol(process)
	return process, err
}

func (p *Parser) buildCompanionForItem(ctx *ParserContext, item *model.Item, pc process.ProcessContent) (Companion, error) {
	slog.Debug("build companions", "module", ctx.CurrentModule(), "item", item.Qualifier)

	companion := make(Companion, 0)

	coItem, err := p.buildCompanionCoItem(item)
	if err != nil {
		return companion, err
	}
	companion = append(companion, coItem)

	coProcess, err := p.buildCompanionCoProcess(item, coItem)
	if err != nil {
		return companion, err
	}
	companion = append(companion, coProcess)

	process, err := p.buildCompanionProcess(item, pc)
	if err != nil {
		return companion, err
	}
	companion = append(companion, process)
	return companion, nil
}
