package hcl

import (
	"fmt"
	"net/url"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/tychonis/cyanotype/internal/digest"
	"github.com/tychonis/cyanotype/model"
)

func (p *Parser) getArtifactDigest(_ *ParserContext, source string) (string, error) {
	parsed, err := url.Parse(source)
	if err != nil {
		return "", err
	}
	switch parsed.Scheme {
	case "file":
		filepath := parsed.Host + parsed.Path
		return digest.SHA256FromFile(filepath)
	case "digest":
		return parsed.Opaque, nil
	case "s3":
		return "", fmt.Errorf("s3 scheme not supported yet")
	default:
		return "", fmt.Errorf("scheme not supported: %s", parsed.Scheme)
	}
}

func (p *Parser) parseArtifactBlock(ctx *ParserContext, block *hclsyntax.Block) (*model.Artifact, error) {
	if len(block.Labels) != 1 {
		return nil, fmt.Errorf(
			"%s: artifact block must have exactly one label",
			block.TypeRange.String(),
		)
	}

	artifact := &model.Artifact{
		Name: block.Labels[0],
	}

	attrs, diags := block.Body.JustAttributes()
	if diags.HasErrors() {
		return nil, diags
	}

	filename, err := getString(attrs, "filename")
	if err != nil {
		return nil, err
	}
	artifact.Filename = filename
	tag, err := getString(attrs, "tag")
	if err != nil {
		return nil, err
	}
	artifact.Tag = tag
	source, err := getString(attrs, "source")
	if err != nil {
		return nil, err
	}
	artifact.Source = source
	artifact.Digest, err = p.getArtifactDigest(ctx, source)
	if err != nil {
		return nil, err
	}
	return artifact, nil
}
