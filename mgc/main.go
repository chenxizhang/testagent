package main

import (
	"github.com/chenxizhang/testagent/mgc/cmd"
)

// Version is the current version of mgc.
// This value is updated by the release process and used to trigger auto-release.
const Version = "0.0.1"

func main() {
	cmd.Execute(Version)
}
