package domain

// SubmoduleState classifies a git submodule's current condition.
type SubmoduleState string

const (
	SubmodulePresent  SubmoduleState = "present"
	SubmoduleMissing  SubmoduleState = "missing"
	SubmoduleOutdated SubmoduleState = "outdated"
	SubmoduleConflict SubmoduleState = "conflict"
)

// Submodule holds runtime data for one git submodule.
type Submodule struct {
	Name   string
	Path   string
	Commit string
	State  SubmoduleState
	Detail string
}
