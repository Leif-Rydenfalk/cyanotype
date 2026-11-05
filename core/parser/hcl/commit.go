package hcl

import (
	"log/slog"

	"github.com/tychonis/cyanotype/core/catalog"
	"github.com/tychonis/cyanotype/model"
)

func (p *Parser) Commit(cat *catalog.Catalog) error {
	return p.commit(cat, false)
}

func (p *Parser) PreviewCommit(cat *catalog.Catalog) error {
	return p.commit(cat, true)
}

func (p *Parser) isSymbolChanged(old model.ConcreteSymbol, new model.ConcreteSymbol) bool {
	if old == nil && new == nil {
		return false
	}
	if old == nil || new == nil {
		return true
	}
	return old.GetDigest() != new.GetDigest()
}

func (p *Parser) commit(cat *catalog.Catalog, dryrun bool) error {
	err := cat.SetModuleName(p.ModuleName)
	if err != nil {
		return err
	}
	revision := cat.NewRevision()
	change := 0
	for qualifier, newSym := range p.Symbols.ConcreteSymbols {
		oldSym, err := cat.FindCurrent(qualifier)
		if err != nil && err != catalog.ErrNotFound {
			return err
		}
		if !p.isSymbolChanged(oldSym, newSym) {
			continue
		}
		if !dryrun {
			err = cat.Add(revision, newSym)
			if err != nil {
				return err
			}
		} else {
			slog.Info("New symbol", "qualifier", qualifier, "digest", newSym.GetDigest())
		}
		change++
	}
	if dryrun || change == 0 {
		return nil
	}
	for _, artifact := range p.Artifacts {
		err := cat.SaveArtifact(artifact)
		if err != nil {
			return err
		}
	}
	return cat.Commit(revision)
}
