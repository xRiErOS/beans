package commands

import _ "embed"

// FindingsSchema is the JSON Schema (draft 2020-12) document defining the
// findings-artifact shape (`{ "findings": [ <record>, ... ] }`) and the
// finding-record shape shared by every reviewer and consumer:
//
//   - `beans promote` (WP2, internal/commands/promote.go) loads this document
//     and validates each selected record against it before writing anything.
//   - `skill:review-spec` / `specs-reviewer` (WP3) write records that conform
//     to it.
//
// This package only publishes the document; it performs no validation and
// chooses no JSON-Schema library (that is WP2's concern).
//
//go:embed findings-schema.json
var FindingsSchema string
