# Cyanotype

**Cyanotype** is an open-source toolchain for managing **Bills of Materials (BoMs) as Code**.
It brings modern software practices—version control, reproducible builds, and declarative configuration—to the world of physical product development.

Cyanotype is part of the [Tychonis](https://tychonis.com) ecosystem, which is building an alternative to traditional PLM systems with openness, modularity, and developer-first ergonomics.

---

## ✨ Why Cyanotype?

- **BoM as Code**
  Author and manage engineering, manufacturing, and service BoMs using a declarative DSL inspired by Terraform/HCL.

- **Unified BoM**
  Keep EBOM, MBOM, and Service BoM consistent in one system—no more disconnected spreadsheets or fragile PLM exports.

- **Versioned & Reproducible**
  Store BoMs in Git, diff across commits, rebuild at any revision, and track supersession/lineage of parts.

- **Extensible**
  Designed to integrate with CAD, supply chain data, manufacturing processes, and even external Git repos.

- **Open Source**
  Unlike closed PLM systems, Cyanotype is transparent, hackable, and community-driven.

---

## ⚠️ Disclaimer

Cyanotype is in a **very early stage** of development.
The DSL, data models, and APIs are subject to rapid iteration and change.
Expect breaking changes until we approach a stable release. We encourage experimentation, feedback, and contributions—but do not rely on Cyanotype yet for production-critical workloads.

# Cyanotype Core Architecture

## Overview

Cyanotype is a **BoM-as-Code** framework for representing engineering knowledge as immutable objects. Unlike traditional PLM systems, a catalog is not a mutable database describing the latest product state. Instead, it is an append-only collection of engineering knowledge that grows over time.

The system is divided into several distinct stages:

```text
Source
    ↓
Compile
    ↓
Revision
    ↓
Integrate
    ↓
Catalog
    ↓
Instantiate
    ↓
Concrete Product
```

Each stage has a single responsibility and produces immutable artifacts.

---

# Core Concepts

## Symbol

A symbol is the fundamental immutable object in Cyanotype.

A symbol may represent an item, process, coitem, coprocess, or any future object type.

Once created, a symbol is never modified.

---

## Revision

A revision is the result of compiling source code.

A revision is independent of any catalog.

Its responsibility is simply to introduce new symbols.

Conceptually,

```text
Revision
    Parent
    New Symbols
```

A revision does **not** perform semantic integration.

It does **not** decide supersession.

It does **not** modify existing catalog knowledge.

It only records what was authored.

---

## Catalog

A catalog is an append-only knowledge base.

It contains:

* every symbol ever introduced
* every revision
* every inferred relationship
* every integration result

Nothing is deleted.

Engineering knowledge accumulates over time.

---

## Integration

Integration is the process of pushing a revision into a catalog.

Unlike Git, push is **not** merely transferring objects.

During integration the system may discover new relationships between newly introduced symbols and existing catalog knowledge.

Examples include:

* identifying interchangeable items
* creating coprocesses
* inferring supersession
* generating additional semantic relationships

These inferred objects belong to the integration step rather than the authored revision.

Future versions may allow engineers to review and edit inferred relationships before they become part of the catalog.

---

## Instantiation

Instantiation is a runtime process.

It takes a build environment and produces a concrete product.

Instantiation performs decisions such as:

* selecting appropriate coprocesses
* resolving interchangeable items
* applying constraints
* producing a concrete product configuration

Instantiation never modifies the catalog.

---

# Item / CoItem

The central abstraction in Cyanotype is separating **identity** from **interchangeability**.

## Item

An Item represents one specific engineering object.

Examples include:

* one exact bolt specification
* one exact PCB
* one exact motor

Items are immutable.

---

## CoItem

A CoItem represents a semantic equivalence class.

Multiple Items may realize the same CoItem.

For example,

```text
Item A
Item B
Item C

     ↓

  CoItem X
```

The exact relationship is determined by coprocesses.

A CoItem does not imply that every Item is equally preferred.

It only represents that they satisfy the same engineering intent.

---

# Process / CoProcess

## Process

A Process transforms CoItems into Items.

```text
CoItem
    ↓
 Process
    ↓
Item
```

Examples include:

* machining
* assembly
* heat treatment
* coating

---

## CoProcess

A CoProcess transforms an Item into a CoItem.

```text
Item
    ↓
CoProcess
    ↓
CoItem
```

This allows the catalog to express engineering interchangeability.

Different Items may reach the same CoItem through different CoProcesses.

The instantiator chooses the most appropriate CoProcess according to available engineering knowledge.

Supersession is therefore **not** a primitive concept.

Instead, supersession is one possible preference between competing CoProcesses.

---

# Workflow

## 1. Author

The engineer edits source files.

Source is intended to remain concise and ergonomic.

Additional engineering relationships should not need to be written explicitly unless desired.

---

## 2. Compile

Compilation produces an immutable Revision.

The revision contains newly introduced symbols.

Compilation is deterministic and independent of any catalog.

---

## 3. Integrate

A revision is pushed into a catalog.

Integration compares the revision against existing knowledge.

The integration process may infer additional symbols such as:

* CoProcesses
* supersession relationships
* future semantic relationships

These become part of the catalog.

---

## 4. Instantiate

Given a build environment, the instantiator resolves:

* Items
* CoItems
* Processes
* CoProcesses

to produce a concrete product.

Different environments may legitimately produce different concrete realizations while sharing the same underlying engineering knowledge.

---

# Design Principles

* Everything meaningful is immutable.
* Catalogs only accumulate knowledge.
* Revisions introduce authored knowledge.
* Integration introduces inferred knowledge.
* Instantiation makes runtime decisions.
* Interchangeability is represented explicitly through CoItems and CoProcesses.
* Supersession is a preference, not a fundamental relationship.
* Source should remain simple enough for mechanical engineers to adopt without needing to understand the underlying semantic framework.


## 🚀 Getting Started

### Installation

Download Cyanotype for your operating system from [GitHub Releases](https://github.com/tychonis/cyanotype/releases).

Extract the archive and add the `cyanotype` executable to your `PATH`.

Verify the installation:

```bash
cyanotype version
```

Alternatively, build from source using Go 1.26 or later:

```bash
git clone https://github.com/tychonis/cyanotype.git
cd cyanotype
go build -o cyanotype .
```

### Example: Pick and Place Machine

The [Pick and Place Machine](https://github.com/tychonis/pnp) example demonstrates how a real mechanical assembly can be represented and managed as code.

Clone the example repository:

```bash
git clone https://github.com/tychonis/pnp.git
cd pnp
```

Generate a Bill of Materials:

```bash
cyanotype bom .pnp
```

By default, the command produces CSV output. To display a readable list instead:

```bash
cyanotype bom .pnp --format list
```

Compile the BPO source into a local catalog:

```bash
cyanotype commit
```

Generate an assembly tree:

```bash
cyanotype tree .pnp
```

This writes the resulting assembly tree to `.pnp.bpc`.

All these operations run locally without requiring a server or authentication.

### Learn More

- [Documentation](https://docs.tychonis.com)
- [Interactive Demo](https://demo.tychonis.com)
- [Tychonis](https://tychonis.com)

**Note:** Cyanotype is under active development. CLI commands, file formats, and data models may change between releases.

## License

Copyright © 2026 Tychonis LLC

Cyanotype is licensed under the Apache License, Version 2.0.
See [LICENSE](LICENSE).
