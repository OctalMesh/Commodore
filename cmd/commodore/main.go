/*
Package main is the entry point for the standalone `commodore` binary.

It uses the Commander facade and the squadron/unit runtime model defined in
examples/configuration.
*/
package main

import (
	"os"

	"github.com/OctalMesh/Commodore/pkg/sdk"
	"github.com/OctalMesh/Commodore/pkg/version"
)

func main() {
	engine := sdk.NewCommander(sdk.Options{
		Version: version.Version,
		Binary:  "commodore",
	})

	if errorValue := engine.Execute(); errorValue != nil {
		engine.Logger().Error("fleet execution failed: %v", errorValue)
		os.Exit(1)
	}
}
