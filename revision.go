package pstackrex

import _ "embed"

// UpstreamRevision is the source commit copied into upstream/.
//
//go:embed upstream/REVISION
var UpstreamRevision string

// SetupTemplate is upstream's role vocabulary, embedded for installed binaries.
//
//go:embed upstream/skills/setup-pstack/SKILL.md
var SetupTemplate string
