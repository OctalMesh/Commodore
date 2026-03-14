package actions

import "github.com/OctalMesh/Commodore/internal/core/ports"

// ModulesStatus runs git submodule status for the runtime root.
func ModulesStatus(engine Engine) error {
	return runGitSubmoduleCommand(engine, "status")
}

// ModulesInit runs git submodule init for the runtime root.
func ModulesInit(engine Engine) error {
	return runGitSubmoduleCommand(engine, "init")
}

// ModulesUpdate runs git submodule update for the runtime root.
func ModulesUpdate(engine Engine) error {
	return runGitSubmoduleCommand(engine, "update", "--recursive")
}

// ModulesSync runs git submodule sync for the runtime root.
func ModulesSync(engine Engine) error {
	return runGitSubmoduleCommand(engine, "sync", "--recursive")
}

func runGitSubmoduleCommand(engine Engine, arguments ...string) error {
	if errorValue := engine.EnsureInitialized(); errorValue != nil {
		return errorValue
	}

	runtimeRoot := engine.Root()
	commandArguments := append([]string{"submodule"}, arguments...)
	request := ports.ProcessRunRequest{Command: "git", Args: commandArguments, Dir: runtimeRoot.Node.GetPath()}
	if runError := engine.RunProcess(request); runError != nil {
		return runError
	}

	return nil
}
