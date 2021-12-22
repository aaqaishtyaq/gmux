package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

type ShellError struct {
	Command string
	Err     error
}

func (e *ShellError) Error() string {
	return fmt.Sprintf("Cannot run %q. Error %v", e.Command, e.Err)
}

type Executor interface {
	Exec(cmd *exec.Cmd) (string, error)
	ExecQuiet(cmd *exec.Cmd) error
}

type DefaultExecutor struct {
	logger *log.Logger
}

func (c DefaultExecutor) Exec(cmd *exec.Cmd) (string, error) {
	if c.logger != nil {
		c.logger.Println(strings.Join(cmd.Args, " "))
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		if c.logger != nil {
			c.logger.Println(err)
		}

		return "", &ShellError{strings.Join(cmd.Args, " "), err}
	}

	return strings.TrimSuffix(string(output), "\n"), nil
}

func (c DefaultExecutor) ExecQuiet(cmd *exec.Cmd) error {
	if c.logger != nil {
		c.logger.Println(strings.Join(cmd.Args, " "))
	}

	err := cmd.Run()
	if err != nil {
		if c.logger != nil {
			c.logger.Println(err)
		}
		return &ShellError{strings.Join(cmd.Args, " "), err}
	}
	return nil
}
