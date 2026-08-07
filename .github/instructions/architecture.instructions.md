---
description: "Use when implementing any part of the scribus-book-generator pipeline: Scribus document generation, layout planning, layout.json schema, Markdown parsing, image handling, AI integration, configuration, or module structure. Covers architecture rules, design contracts, and incremental build strategy."
---
# Copilot Instructions — AI-Assisted Book Publishing

This repository builds an automated picture-book publishing pipeline using Go and Scribus.

## Core Architecture

The system has two distinct phases:

1. AI-assisted layout planning
2. Deterministic Scribus document generation

The pipeline is:

```
Markdown + images → AI analysis → layout.json → Scribus Go generator → editable .sla → manual review → press-ready PDF
```

## Critical Design Rule

`layout.json` is the contract between the AI system and the Scribus renderer.

The AI may recommend layout decisions, but it must never directly construct the Scribus document.

The Scribus generator must operate deterministically from:

- book configuration
- Markdown
- `layout.json`
- image files

Identical inputs should produce essentially identical Scribus documents.

## Scribus

The final generated artifact must be an editable `.sla` Scribus document.

Do not design the system around direct PDF generation.

Scribus-specific APIs and logic should be isolated from general parsing, configuration, validation, and layout-planning code.

Most application logic should be testable without launching Scribus.

## Content

Book chapters are stored as Markdown files.

Images are organized by chapter.

Markdown parsing should assign stable paragraph IDs so image placement does not depend solely on paragraph numbers.

## Layout

The layout system should support:

- inline images
- left/right floated images
- text wrapping
- captions
- full-page images
- chapter-opening images
- page breaks
- two-page spreads
- image priority
- configurable image sizing
- optional cropping

Layout instructions are stored in `layout.json`.

Validate layout data before Scribus generation. Invalid image paths, paragraph references, dimensions, or incompatible layout instructions should produce clear errors.

## Configuration

Do not hard-code:

- page dimensions
- margins
- bleed
- fonts
- font sizes
- leading
- image borders
- printer requirements

Store these in configuration files. Printer specifications will be added later.

## Code Organization

Keep separate modules for:

- Markdown parsing
- image metadata/analysis
- AI layout planning
- layout validation
- configuration
- Scribus generation

Avoid monolithic binaries. Prefer straightforward Go and minimal dependencies. Individual pipeline stages can be built as small CLI commands runnable as scripts.

## Development Strategy

Build incrementally. The first milestone should **not** use AI.

First prove that:

```
Markdown + hand-written layout.json + images → Scribus → editable .sla
```

The initial Scribus document should support:

- chapter title
- flowing body text
- paragraph styles
- one image
- text wrapping
- caption

Only after Scribus generation works reliably should AI-generated layout instructions be added.

## Documentation

Architectural decisions should be documented under `docs/`.

When implementing a major feature, consult the relevant documents there rather than duplicating architecture rules in source files.

If an implementation conflicts with these architectural rules, stop and explain the conflict before changing the architecture.
