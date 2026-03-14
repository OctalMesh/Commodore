/*
Package main is the entry point for a custom CLI built on top of the Commodore
SDK. This example demonstrates how to bootstrap the Commander, extend it with
custom Go logic, and hook into the naval-inspired lifecycle (Squadrons, Units,
Signals).
*/
package main

import (
	"os"

	"github.com/OctalMesh/Commodore/pkg/sdk"
)

// Version info injected at build time.
var Version = "v1.0.0"

func main() {
	// 1. Initialize the Commander Engine

	engine := sdk.NewCommander(sdk.Options{
		Version: Version,
		Binary:  "custom", // Default binary name of custom CLI
	})

	// 2. Custom Commands

	// While configuration allows defining shell 'maneuvers', sometimes you need
	// complex, interactive, or purely Go-based logic.
	engine.AddCommand(&sdk.Command{
		Use:   "seed",
		Short: "Seed the platform database with initial test data",
		RunE: func(context sdk.Context) error {
			context.Log().Info("Connecting to PostgreSQL...")
			// Your custom Go logic here...
			context.Log().Success("Database successfully seeded.")

			return nil
		},
	})

	// 3. Execution

	// Execute parses os.Args, resolves the manifest DAG, and runs the requested
	// command.
	if err := engine.Execute(); err != nil {
		engine.Logger().Error("Fleet execution failed: %v", err)
		os.Exit(1)
	}
}
