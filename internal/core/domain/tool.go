package domain

// Tool describes an external dependency required or recommended by OctalWeb.
type Tool struct {
	// Name is the human-readable name shown in CLI output.
	Name string
	// Cmd is the executable name to locate in PATH.
	Cmd string
	// VersionArgs are passed to Cmd to retrieve its version string.
	VersionArgs []string
	// Required marks tools that must be present for OctalWeb to function.
	Required bool
	// InstallURL is shown when the tool is missing.
	InstallURL string
}

// ToolResult holds the outcome of checking a single tool.
type ToolResult struct {
	Tool    Tool
	Found   bool
	Version string
}
