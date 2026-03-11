/*
Package submodule implements ports.SubmoduleRepository using the git CLI.
*/
package submodule

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/OctalMesh/Commodore/internal/core/domain"
)

// Repository implements ports.SubmoduleRepository.
type Repository struct {}

// New returns a Repository.
func New() *Repository { return &Repository{} }

// List returns the current state of all direct git submodules.
func (r *Repository) List(repoRoot string) ([]domain.Submodule, error) {
	cmd := exec.Command("git", "-C", repoRoot, "submodule", "status")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git submodule status: %w", err)
	}
	return parse(out), nil
}

// Init runs "git submodule update --init --recursive".
func (r *Repository) Init(repoRoot string) error {
	return inherit(repoRoot, "git", "submodule", "update", "--init", "--recursive")
}

// Update runs "git submodule update --remote --merge".
func (r *Repository) Update(repoRoot string) error {
	return inherit(repoRoot, "git", "submodule", "update", "--remote", "--merge")
}

// Sync runs "git submodule sync --recursive".
func (r *Repository) Sync(repoRoot string) error {
	return inherit(repoRoot, "git", "submodule", "sync", "--recursive")
}

func inherit(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// Line format: [prefix]SHA path [(description)].
func parse(out []byte) []domain.Submodule {
	var mods []domain.Submodule
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		if len(line) < 2 {
			continue
		}
		prefix := line[0]
		rest := strings.TrimSpace(line[1:])
		parts := strings.Fields(rest)
		if len(parts) < 2 {
			continue
		}
		commit := parts[0]
		path := parts[1]
		detail := ""
		if len(parts) > 2 {
			detail = strings.Join(parts[2:], " ")
		}
		state := domain.SubmodulePresent
		switch prefix {
		case '-':
			state = domain.SubmoduleMissing
		case '+':
			state = domain.SubmoduleOutdated
		case 'U':
			state = domain.SubmoduleConflict
		}
		if len(commit) > 8 {
			commit = commit[:8]
		}
		name := path
		if idx := strings.LastIndex(path, "/"); idx >= 0 {
			name = path[idx+1:]
		}
		mods = append(mods, domain.Submodule{
			Name:   name,
			Path:   path,
			Commit: commit,
			State:  state,
			Detail: detail,
		})
	}
	return mods
}
