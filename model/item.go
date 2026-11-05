package model

import (
	"github.com/tychonis/cyanotype/internal/stable"
)

type ItemID = Digest
type LinkType string

type ItemBase struct {
	Qualifier string `json:"-" yaml:"-"`
	Digest    ItemID `json:"-" yaml:"-"`

	Type    string       `json:"type" yaml:"type"`
	Content *ItemContent `json:"content" yaml:"content"`
}

// Item corresponds to a single immutable snapshot of a part or assembly.
// Any change to spec, composition, process or metadata produces a new Item.
// Items are linked through Predecessor for traceability supersession or interchangeability.
type Item struct {
	ItemBase
	Implement []ContractID `json:"implement" yaml:"implement"`
}

// CoItem defines requirements.
type CoItem struct {
	ItemBase
	Require []ContractID `json:"require" yaml:"require"`
}

type ItemContent struct {
	Name       string      `json:"name" yaml:"name"`
	Source     string      `json:"source,omitempty" yaml:"source,omitempty"`
	PartNumber string      `json:"part_number" yaml:"part_number"`
	Artifacts  []*Artifact `json:"artifacts,omitempty" yaml:"artifacts,omitempty"`
	Details    stable.Map  `json:"details" yaml:"details"`
}

func (i *Item) GetQualifier() string {
	return i.Qualifier
}

func (i *Item) GetDigest() string {
	return i.Digest
}

func (i *Item) GetType() string {
	return i.Type
}

func (i *Item) GetName() string {
	if i.Content != nil {
		return i.Content.Name
	}
	return i.GetQualifier()
}

func (ci *CoItem) GetQualifier() string {
	return ci.Qualifier
}

func (ci *CoItem) GetDigest() string {
	return ci.Digest
}

func (ci *CoItem) GetType() string {
	return ci.Type
}

func (ci *CoItem) GetName() string {
	if ci.Content != nil {
		return ci.Content.Name
	}
	return ci.GetQualifier()
}
