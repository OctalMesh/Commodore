/*
Package toolcheck implements ports.ToolChecker by probing executables on PATH.
*/
package toolcheck

import (
	"os/exec"
	"runtime"
	"strings"

	"github.com/OctalMesh/Commodore/internal/core/domain"
)

// Checker implements ports.ToolChecker.
type Checker struct {
	tools []domain.Tool
}

// New returns a Checker pre-loaded with the canonical OctalWeb tool list.
func New() *Checker { return &Checker{tools: defaultTools()} }

// CheckAll runs all checks and returns results in the same order as the tool list.
func (c *Checker) CheckAll() []domain.ToolResult {
	results := make([]domain.ToolResult, len(c.tools))
	for i, t := range c.tools {
		results[i] = checkOne(t)
	}
	return results
}

func checkOne(t domain.Tool) domain.ToolResult {
	if _, err := exec.LookPath(t.Cmd); err != nil {
		return domain.ToolResult{Tool: t, Found: false}
	}
	out, _ := exec.Command(t.Cmd, t.VersionArgs...).CombinedOutput()
	ver := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]
	if len(ver) > 70 {
		ver = ver[:70]
	}
	return domain.ToolResult{Tool: t, Found: true, Version: ver}
}

func defaultTools() []domain.Tool {
	return []domain.Tool{
		{Name: "Git", Cmd: "git", VersionArgs: []string{"--version"}, Required: true, InstallURL: "https://git-scm.com/downloads"},
		{Name: "Go", Cmd: "go", VersionArgs: []string{"version"}, Required: true, InstallURL: "https://go.dev/dl/"},
		{Name: "Docker", Cmd: "docker", VersionArgs: []string{"version", "--format", "{{.Client.Version}}"}, Required: true, InstallURL: dockerInstallURL()},
		{Name: "Java", Cmd: "java", VersionArgs: []string{"-version"}, Required: true, InstallURL: "https://aws.amazon.com/corretto/"},
		{Name: "Node.js", Cmd: "node", VersionArgs: []string{"--version"}, Required: false, InstallURL: "https://nodejs.org/en/download/"},
		{Name: "npm", Cmd: "npm", VersionArgs: []string{"--version"}, Required: false, InstallURL: "included with Node.js"},
		{Name: "Tilt", Cmd: "tilt", VersionArgs: []string{"version"}, Required: true, InstallURL: "https://docs.tilt.dev/install.html"},
	}
}

func dockerInstallURL() string {
	switch runtime.GOOS {
	case "windows":
		return "https://docs.docker.com/desktop/install/windows-install/"
	case "darwin":
		return "https://docs.docker.com/desktop/install/mac-install/"
	default:
		return "https://docs.docker.com/engine/install/"
	}
}
