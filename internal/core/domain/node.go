package domain

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// NodeRole defines supported orchestration node roles.
type NodeRole string

const (
	// NodeRoleSquadron is a recursive orchestrator node.
	NodeRoleSquadron NodeRole = "squadron"
	// NodeRoleUnit is an atomic leaf execution node.
	NodeRoleUnit NodeRole = "unit"
)

// ReactorProvider identifies the runtime execution provider.
type ReactorProvider string

const (
	// ReactorProviderTilt identifies Tilt-driven execution.
	ReactorProviderTilt ReactorProvider = "tilt"
	// ReactorProviderNative identifies direct command execution.
	ReactorProviderNative ReactorProvider = "native"
)

// ValidationError indicates invalid domain data.
type ValidationError struct {
	Message string
	Cause   error
}

func (errorValue *ValidationError) Error() string {
	if errorValue == nil {
		return "validation error"
	}
	if errorValue.Cause == nil {
		return errorValue.Message
	}
	return fmt.Sprintf("%s: %v", errorValue.Message, errorValue.Cause)
}

func (errorValue *ValidationError) Unwrap() error {
	if errorValue == nil {
		return nil
	}
	return errorValue.Cause
}

// DiscoveryError indicates configuration discovery failures.
type DiscoveryError struct {
	Message string
	Cause   error
}

func (errorValue *DiscoveryError) Error() string {
	if errorValue == nil {
		return "discovery error"
	}
	if errorValue.Cause == nil {
		return errorValue.Message
	}
	return fmt.Sprintf("%s: %v", errorValue.Message, errorValue.Cause)
}

func (errorValue *DiscoveryError) Unwrap() error {
	if errorValue == nil {
		return nil
	}
	return errorValue.Cause
}

// DependencyError indicates dependency graph issues.
type DependencyError struct {
	Message string
	Cause   error
}

func (errorValue *DependencyError) Error() string {
	if errorValue == nil {
		return "dependency error"
	}
	if errorValue.Cause == nil {
		return errorValue.Message
	}
	return fmt.Sprintf("%s: %v", errorValue.Message, errorValue.Cause)
}

func (errorValue *DependencyError) Unwrap() error {
	if errorValue == nil {
		return nil
	}
	return errorValue.Cause
}

// SignalRoutingError indicates failures while routing a signal command.
type SignalRoutingError struct {
	Message string
	Cause   error
}

func (errorValue *SignalRoutingError) Error() string {
	if errorValue == nil {
		return "signal routing error"
	}
	if errorValue.Cause == nil {
		return errorValue.Message
	}
	return fmt.Sprintf("%s: %v", errorValue.Message, errorValue.Cause)
}

func (errorValue *SignalRoutingError) Unwrap() error {
	if errorValue == nil {
		return nil
	}
	return errorValue.Cause
}

// ManeuverDefinition declares an executable maneuver.
type ManeuverDefinition struct {
	Call        string   `yaml:"call"`
	Description string   `yaml:"description"`
	Action      []string `yaml:"action"`
}

// ReactorBlueprint maps an environment to a path or inline action.
type ReactorBlueprint struct {
	EnvironmentName string   `yaml:"env"`
	Path            string   `yaml:"path"`
	Action          []string `yaml:"action"`
}

// EnvironmentDefinition defines inherited environment files and variables.
type EnvironmentDefinition struct {
	Name      string   `yaml:"name"`
	Files     []string `yaml:"files"`
	Variables []string `yaml:"variables"`
}

// ReactorDefinition configures node runtime behavior.
type ReactorDefinition struct {
	Provider     ReactorProvider         `yaml:"provider"`
	Network      string                  `yaml:"network"`
	ContextPath  string                  `yaml:"context"`
	Blueprints   []ReactorBlueprint      `yaml:"blueprints"`
	Environments []EnvironmentDefinition `yaml:"environments"`
}

// HealthCheckDefinition defines subordinate readiness checks.
type HealthCheckDefinition struct {
	Test     []string      `yaml:"test"`
	Interval time.Duration `yaml:"interval"`
	Timeout  time.Duration `yaml:"timeout"`
	Retries  int           `yaml:"retries"`
}

// NodeReference declares one subordinate entry in a squadron manifest.
type NodeReference struct {
	ID          string                 `yaml:"id"`
	Path        string                 `yaml:"path"`
	Tags        []string               `yaml:"tags"`
	After       []string               `yaml:"after"`
	HealthCheck *HealthCheckDefinition `yaml:"healthcheck"`
}

// NodeConfiguration is a YAML representation for squadron and unit configs.
type NodeConfiguration struct {
	Role      NodeRole             `yaml:"role"`
	ID        string               `yaml:"id"`
	Binary    string               `yaml:"binary"`
	Reactor   ReactorDefinition    `yaml:"reactor"`
	Manifest  []NodeReference      `yaml:"manifest"`
	Maneuvers []ManeuverDefinition `yaml:"maneuvers"`
}

// OrchestrationNode is the common node behavior contract.
type OrchestrationNode interface {
	GetID() string
	GetRole() NodeRole
	GetPath() string
	GetTags() []string
	GetDependencies() []string
	GetSubordinates() []OrchestrationNode
	ResolveSubordinateByID(subordinateID string) (OrchestrationNode, error)
	CollectDescendantNodes() []OrchestrationNode
	BuildEffectiveEnvironmentByName(
		environmentName string,
		parentEnvironment EnvironmentDefinition,
	) (EnvironmentDefinition, error)
	ExecuteManeuverByName(maneuverName string) ([]string, error)
}

// SquadronNode models a recursive orchestration node.
type SquadronNode struct {
	ID           string
	Path         string
	Tags         []string
	After        []string
	Reactor      ReactorDefinition
	Maneuvers    []ManeuverDefinition
	Subordinates []OrchestrationNode
}

// UnitNode models a leaf execution node.
type UnitNode struct {
	ID        string
	Path      string
	Tags      []string
	After     []string
	Reactor   ReactorDefinition
	Maneuvers []ManeuverDefinition
}

func (squadron *SquadronNode) GetID() string { return squadron.ID }

func (squadron *SquadronNode) GetRole() NodeRole { return NodeRoleSquadron }

func (squadron *SquadronNode) GetPath() string { return squadron.Path }

func (squadron *SquadronNode) GetTags() []string { return append([]string(nil), squadron.Tags...) }

func (squadron *SquadronNode) GetDependencies() []string {
	return append([]string(nil), squadron.After...)
}

func (squadron *SquadronNode) GetSubordinates() []OrchestrationNode {
	return append([]OrchestrationNode(nil), squadron.Subordinates...)
}

func (squadron *SquadronNode) ResolveSubordinateByID(subordinateID string) (OrchestrationNode, error) {
	for _, subordinate := range squadron.Subordinates {
		if subordinate.GetID() == subordinateID {
			return subordinate, nil
		}
	}

	return nil, &SignalRoutingError{
		Message: fmt.Sprintf("subordinate %q not found in squadron %q", subordinateID, squadron.ID),
	}
}

func (squadron *SquadronNode) CollectDescendantNodes() []OrchestrationNode {
	descendants := make([]OrchestrationNode, 0, len(squadron.Subordinates))
	for _, subordinate := range squadron.Subordinates {
		descendants = append(descendants, subordinate)
		descendants = append(descendants, subordinate.CollectDescendantNodes()...)
	}

	return descendants
}

func (squadron *SquadronNode) BuildEffectiveEnvironmentByName(
	environmentName string,
	parentEnvironment EnvironmentDefinition,
) (EnvironmentDefinition, error) {
	return buildEffectiveEnvironmentByName(environmentName, parentEnvironment, squadron.Reactor.Environments)
}

func (squadron *SquadronNode) ExecuteManeuverByName(maneuverName string) ([]string, error) {
	return executeManeuverByName(squadron.ID, squadron.Maneuvers, maneuverName)
}

func (unit *UnitNode) GetID() string { return unit.ID }

func (unit *UnitNode) GetRole() NodeRole { return NodeRoleUnit }

func (unit *UnitNode) GetPath() string { return unit.Path }

func (unit *UnitNode) GetTags() []string { return append([]string(nil), unit.Tags...) }

func (unit *UnitNode) GetDependencies() []string { return append([]string(nil), unit.After...) }

func (unit *UnitNode) GetSubordinates() []OrchestrationNode { return nil }

func (unit *UnitNode) ResolveSubordinateByID(subordinateID string) (OrchestrationNode, error) {
	return nil, &SignalRoutingError{
		Message: fmt.Sprintf("unit %q has no subordinate %q", unit.ID, subordinateID),
	}
}

func (unit *UnitNode) CollectDescendantNodes() []OrchestrationNode { return nil }

func (unit *UnitNode) BuildEffectiveEnvironmentByName(
	environmentName string,
	parentEnvironment EnvironmentDefinition,
) (EnvironmentDefinition, error) {
	return buildEffectiveEnvironmentByName(environmentName, parentEnvironment, unit.Reactor.Environments)
}

func (unit *UnitNode) ExecuteManeuverByName(maneuverName string) ([]string, error) {
	return executeManeuverByName(unit.ID, unit.Maneuvers, maneuverName)
}

// DiscoverConfigurationPathsRecursively scans for supported node configuration files.
func DiscoverConfigurationPathsRecursively(rootPath string) ([]string, error) {
	absoluteRootPath, errorValue := filepath.Abs(rootPath)
	if errorValue != nil {
		return nil, &DiscoveryError{
			Message: fmt.Sprintf("resolving root path %q", rootPath),
			Cause:   errorValue,
		}
	}

	matchingPaths := make([]string, 0)
	configurationNames := map[string]struct{}{
		".commodore":      {},
		".commodore.yaml": {},
		".commodore.yml":  {},
	}

	walkError := filepath.WalkDir(absoluteRootPath, func(path string, directoryEntry fs.DirEntry, walkError error) error {
		if walkError != nil {
			return walkError
		}

		if directoryEntry.IsDir() {
			if strings.EqualFold(directoryEntry.Name(), ".git") {
				return filepath.SkipDir
			}
			return nil
		}

		if _, found := configurationNames[directoryEntry.Name()]; found {
			matchingPaths = append(matchingPaths, path)
		}

		return nil
	})
	if walkError != nil {
		return nil, &DiscoveryError{
			Message: fmt.Sprintf("scanning configuration files under %q", absoluteRootPath),
			Cause:   walkError,
		}
	}

	sort.Strings(matchingPaths)
	return matchingPaths, nil
}

func buildEffectiveEnvironmentByName(
	environmentName string,
	parentEnvironment EnvironmentDefinition,
	localEnvironments []EnvironmentDefinition,
) (EnvironmentDefinition, error) {
	mergedEnvironment := EnvironmentDefinition{
		Name:      environmentName,
		Files:     append([]string(nil), parentEnvironment.Files...),
		Variables: append([]string(nil), parentEnvironment.Variables...),
	}

	for _, localEnvironment := range localEnvironments {
		if localEnvironment.Name != environmentName {
			continue
		}

		mergedEnvironment.Files = mergeFileLists(parentEnvironment.Files, localEnvironment.Files)

		resolvedVariables, errorValue := mergeVariablesByName(parentEnvironment.Variables, localEnvironment.Variables)
		if errorValue != nil {
			return EnvironmentDefinition{}, errorValue
		}
		mergedEnvironment.Variables = resolvedVariables

		return mergedEnvironment, nil
	}

	return mergedEnvironment, nil
}

func executeManeuverByName(
	nodeIdentifier string,
	maneuvers []ManeuverDefinition,
	maneuverName string,
) ([]string, error) {
	for _, maneuver := range maneuvers {
		if maneuver.Call != maneuverName {
			continue
		}

		if len(maneuver.Action) == 0 {
			return nil, &ValidationError{
				Message: fmt.Sprintf("maneuver %q in node %q has empty action", maneuverName, nodeIdentifier),
			}
		}

		return append([]string(nil), maneuver.Action...), nil
	}

	return nil, &ValidationError{Message: fmt.Sprintf("maneuver %q was not found in node %q", maneuverName, nodeIdentifier)}
}

func mergeFileLists(parentFiles []string, localFiles []string) []string {
	seenFiles := make(map[string]struct{}, len(parentFiles)+len(localFiles))
	resultFiles := make([]string, 0, len(parentFiles)+len(localFiles))

	for _, filePath := range parentFiles {
		if _, found := seenFiles[filePath]; found {
			continue
		}
		seenFiles[filePath] = struct{}{}
		resultFiles = append(resultFiles, filePath)
	}

	for _, filePath := range localFiles {
		if _, found := seenFiles[filePath]; found {
			continue
		}
		seenFiles[filePath] = struct{}{}
		resultFiles = append(resultFiles, filePath)
	}

	return resultFiles
}

func mergeVariablesByName(parentVariables []string, localVariables []string) ([]string, error) {
	mergedVariables := make(map[string]string, len(parentVariables)+len(localVariables))
	orderedKeys := make([]string, 0, len(parentVariables)+len(localVariables))

	for _, variableEntry := range parentVariables {
		key, value, errorValue := parseVariableEntry(variableEntry)
		if errorValue != nil {
			return nil, errorValue
		}

		if _, found := mergedVariables[key]; !found {
			orderedKeys = append(orderedKeys, key)
		}
		mergedVariables[key] = value
	}

	for _, variableEntry := range localVariables {
		key, value, errorValue := parseVariableEntry(variableEntry)
		if errorValue != nil {
			return nil, errorValue
		}

		if _, found := mergedVariables[key]; !found {
			orderedKeys = append(orderedKeys, key)
		}
		mergedVariables[key] = value
	}

	resultEntries := make([]string, 0, len(orderedKeys))
	for _, key := range orderedKeys {
		resultEntries = append(resultEntries, key+"="+mergedVariables[key])
	}

	return resultEntries, nil
}

func parseVariableEntry(variableEntry string) (string, string, error) {
	equalsIndex := strings.Index(variableEntry, "=")
	if equalsIndex <= 0 {
		return "", "", &ValidationError{Message: fmt.Sprintf("invalid variable entry %q; expected KEY=VALUE", variableEntry)}
	}

	key := variableEntry[:equalsIndex]
	value := variableEntry[equalsIndex+1:]
	if strings.TrimSpace(key) == "" {
		return "", "", &ValidationError{Message: fmt.Sprintf("invalid variable entry %q; key cannot be empty", variableEntry)}
	}

	return key, value, nil
}

// ResolveConfigurationPath returns the best config file under a directory.
func ResolveConfigurationPath(path string) (string, error) {
	pathInfo, errorValue := os.Stat(path)
	if errorValue != nil {
		return "", &DiscoveryError{Message: fmt.Sprintf("reading path %q", path), Cause: errorValue}
	}

	if !pathInfo.IsDir() {
		return path, nil
	}

	candidates := []string{
		filepath.Join(path, ".commodore"),
		filepath.Join(path, ".commodore.yaml"),
		filepath.Join(path, ".commodore.yml"),
	}

	for _, candidate := range candidates {
		if _, statError := os.Stat(candidate); statError == nil {
			return candidate, nil
		}
	}

	return "", &DiscoveryError{Message: fmt.Sprintf("no Commodore configuration found under %q", path)}
}
