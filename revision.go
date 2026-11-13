package pstackrex

import _ "embed"

// UpstreamRevision is the source commit copied into upstream/.
//
//go:embed upstream/REVISION
var UpstreamRevision string
