package executor

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/zenv/v2/pkg/model"
)

func NewDefaultExecutor() ExecuteFunc {
	return func(cmd string, args []string, envVars []*model.EnvVar) (int, error) {
		command := exec.Command(cmd, args...)

		// Set environment variables
		env := os.Environ()
		for _, envVar := range envVars {
			env = append(env, fmt.Sprintf("%s=%s", envVar.Name, envVar.Value))
		}
		command.Env = env

		// Set up standard streams
		command.Stdin = os.Stdin
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr

		err := command.Run()
		if err != nil {
			// Extract exit code
			if exitError, ok := err.(*exec.ExitError); ok {
				if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
					return status.ExitStatus(), nil
				}
			}
			return 1, goerr.Wrap(err, "failed to execute command")
		}

		return 0, nil
	}
}
