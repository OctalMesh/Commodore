/*
Package main is the entry point for the standalone `commodore` binary.

It reads .commodore in the current working directory, determines the role
(foreman / brigadier / worker), and delegates to the corresponding SDK
factory. This lets a single binary serve all three roles depending on
where it is invoked from.
*/
package main

import (
	"fmt"
	"os"

	"github.com/OctalMesh/Commodore/internal/adapters/config"
	"github.com/OctalMesh/Commodore/internal/core/domain"
	"github.com/OctalMesh/Commodore/internal/roles/brigadier"
	"github.com/OctalMesh/Commodore/internal/roles/foreman"
	"github.com/OctalMesh/Commodore/internal/roles/worker"
	"github.com/OctalMesh/Commodore/pkg/version"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "  x  could not determine working directory:", err)
		os.Exit(1)
	}

	cfg, err := config.New().Load(cwd + "/.commodore")
	if err != nil {
		fmt.Fprintln(os.Stderr, "  x  could not load .commodore:", err)
		os.Exit(1)
	}

	switch cfg.Role {
	case domain.RoleForeman:
		foreman.New(cfg, version.Version).Run()
	case domain.RoleBrigadier:
		brigadier.New(cfg, version.Version).Run()
	case domain.RoleWorker:
		worker.New(cfg, version.Version).Run()
	default:
		fmt.Fprintf(os.Stderr, "  x  unknown role %q in .commodore (expected foreman|brigadier|worker)\n", cfg.Role)
		os.Exit(1)
	}
}
