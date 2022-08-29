//go:build tools
// +build tools

package tools

// tools.go is not meant to be compiled with the project.  It's solely
// here to pin some dependencies on tools we use in the build.
// go modules will include version management and vendoring of
// these tools, even though this source file has a build tag that
// isn't include by the real build.

import _ "golang.org/x/tools/cmd/cover"
