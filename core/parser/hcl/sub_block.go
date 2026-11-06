package hcl

import (
	"fmt"
	"log/slog"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/tychonis/cyanotype/model"
)

type SubBlockContent struct {
	Artifacts []*model.Artifact
}

func (p *Parser) ProcessSubBlocks(ctx *ParserContext, block *hclsyntax.Block) (*SubBlockContent, error) {
	if block == nil {
		return nil, fmt.Errorf("block is nil")
	}

	content := &SubBlockContent{
		Artifacts: []*model.Artifact{},
	}

	for _, child := range block.Body.Blocks {
		switch child.Type {
		case "artifact":
			artifact, err := p.processArtifactBlock(ctx, child)
			if err != nil {
				return nil, err
			}
			p.Artifacts[artifact.Digest] = artifact
			content.Artifacts = append(content.Artifacts, artifact)
		case "contract":
			_, err := p.processContractBlock(ctx, child)
			if err != nil {
				return nil, err
			}
		case "operation":
			err := p.processOperationBlock(ctx, child)
			if err != nil {
				return nil, err
			}
		default:
			slog.Warn("Unrecognized sub block, skipping", slog.String("type", child.Type))
			continue
		}
	}
	return content, nil
}
