package hcl

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/tychonis/cyanotype/internal/cerror"
	"github.com/tychonis/cyanotype/internal/symbols"
	"github.com/tychonis/cyanotype/model"
)

const EXTENSION string = ".bpo"
const IMPLICIT string = "implicit."
const DEFAULT string = "default"

type ParserOptions struct {
	IgnoreArtifacts bool
}

type Parser struct {
	ModuleName string
	Symbols    *symbols.SymbolTable
	Artifacts  map[model.Digest]*model.Artifact

	Options *ParserOptions
}

type ParserContext struct {
	ImportStack []string
}

func NewParserContext() *ParserContext {
	return &ParserContext{
		ImportStack: []string{"."},
	}
}

func (ctx *ParserContext) Import(path string) (*ParserContext, error) {
	for _, existed := range ctx.ImportStack {
		if existed == path {
			return nil, errors.New("cyclic import detected")
		}
	}
	return &ParserContext{
		ImportStack: append([]string{path}, ctx.ImportStack...),
	}, nil
}

func (ctx *ParserContext) CurrentModule() string {
	return ctx.ImportStack[0]
}

func (ctx *ParserContext) NameToQualifier(name string) string {
	prefix := ctx.CurrentModule()
	if prefix == "." {
		prefix = ""
	}
	return prefix + "." + name
}

func NewParser() *Parser {
	return &Parser{
		Symbols:   symbols.NewSymbolTable(),
		Artifacts: make(map[model.Digest]*model.Artifact),
		Options:   &ParserOptions{},
	}
}

func (p *Parser) Resolve(ctx *ParserContext, ref []string) (model.Resolver, error) {
	slog.Debug("Resolving ref", "module", ctx.CurrentModule(), "ref", ref)
	mod, ok := p.Symbols.Modules[ref[0]]
	if ok {
		return mod.Resolve(ref[1:])
	}
	mod, ok = p.Symbols.Modules[ctx.CurrentModule()]
	if !ok {
		return nil, errors.New("no registered symbols")
	}
	return mod.Resolve(ref)
}

func (p *Parser) parseFolder(ctx *ParserContext, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == EXTENSION {
			err = p.parseFile(ctx, filepath.Join(dir, entry.Name()))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *Parser) ParseFolder(dir string) error {
	ctx := NewParserContext()
	return p.parseFolder(ctx, dir)
}

func (p *Parser) parseFile(ctx *ParserContext, filename string) error {
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCLFile(filename)
	if diags.HasErrors() {
		slog.Error("Failed to parse file.", "error", diags.Error())
		return diags
	}

	content, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		slog.Error("Failed to parse content.")
		return errors.New("failed to parse content")
	}

	for _, block := range content.Blocks {
		err := p.registerBlock(ctx, block)
		if err != nil {
			slog.Warn("Error parsing block.", "error", err)
		}
	}
	return nil
}

func (p *Parser) ParseFile(filename string) error {
	ctx := NewParserContext()
	return p.parseFile(ctx, filename)
}

func (p *Parser) Parse(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return p.ParseFolder(path)
	}
	return p.ParseFile(path)
}

func PrintParserError(err error) {
	if err == nil {
		return
	}
	var errWithRange *cerror.ErrWithRange
	if errors.As(err, &errWithRange) {
		r := errWithRange.Range
		var errKeyNotFound *cerror.ErrKeyNotFound
		var msg string
		if errors.As(errWithRange.Err, &errKeyNotFound) {
			msg = fmt.Sprintf("Missing required key: %s", errKeyNotFound.Key)
		} else {
			msg = errWithRange.Err.Error()
		}
		slog.Error(fmt.Sprintf("%s:%d %s", r.Filename, r.Start.Line, msg))
	} else {
		slog.Error("Parser error.", "error", err)
	}
}

func (p *Parser) build(path string) error {
	err := p.Parse(path)
	if err != nil {
		return err
	}
	return p.processModules()
}

func (p *Parser) Build(path string) error {
	err := p.build(path)
	PrintParserError(err)
	return err
}

func (p *Parser) processModules() error {
	for _, module := range p.Symbols.Modules {
		err := p.processModuleScope(module)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) processModuleScope(m *symbols.ModuleScope) error {
	for _, symbol := range m.Symbols {
		switch s := symbol.(type) {
		case *symbols.ModuleScope:
			err := p.processModuleScope(s)
			if err != nil {
				return err
			}
		case *UnprocessedSymbol:
			slog.Debug("Process item", "item", s)
			_, err := p.ProcessSymbol(s)
			if err != nil {
				return err
			}
		default:
		}
	}
	return nil
}

func (p *Parser) resolveBOMLineRef(ctx *ParserContext, ref Ref) (*model.Item, error) {
	qualifier := refToQualifier(ctx, ref)
	itemSym, err := p.Symbols.FindConcreteSymbol(qualifier)
	if err != nil {
		if err != symbols.ErrNotFound {
			return nil, err
		} else {
			sym, err := p.Resolve(ctx, ref)
			if err != nil {
				return nil, err
			}
			unprocessed, ok := sym.(*UnprocessedSymbol)
			if !ok {
				return nil, errors.New("wrong symbol type")
			}
			itemSym, err = p.ProcessSymbol(unprocessed)
			if err != nil {
				return nil, err
			}
			unprocessed.qualifier = itemSym.GetQualifier()
		}
	}

	item, ok := itemSym.(*model.Item)
	if !ok {
		return nil, errors.New("incorrect ref")
	}
	return item, nil
}
