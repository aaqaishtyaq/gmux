package executor

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	switch os.Getenv("TEST_MAIN") {
	case "":
		os.Exit(m.Run())
	case "echo":
		fmt.Println(strings.Join(os.Args[1:], " "))
	case "exit":
		os.Exit(1)
	}
}

func TestExec(t *testing.T) {
	logger := log.New(bytes.NewBuffer([]byte{}), "", 0)
	executor := DefaultExecutor{logger}

	cmd := exec.Command(os.Args[0], "1")
	cmd.Env = append(os.Environ(), "TEST_MAIN=echo")

	output, err := executor.Exec(cmd)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	if output != "1" {
		t.Errorf("expected 1, got %q", output)
	}
}

func TestExecError(t *testing.T) {
	logger := log.New(bytes.NewBuffer([]byte{}), "", 0)
	executor := DefaultExecutor{logger}

	cmd := exec.Command(os.Args[0], "1")
	cmd.Env = append(os.Environ(), "TEST_MAIN=exit")

	_, err := executor.Exec(cmd)
	if err == nil {
		t.Errorf("expected error")
	}

	got := cmd.ProcessState.ExitCode()
	if got != 1 {
		t.Errorf("expected %d, got %d", 1, got)
	}
}

func TestExecQuiet(t *testing.T) {
	logger := log.New(bytes.NewBuffer([]byte{}), "", 0)
	executor := DefaultExecutor{logger}

	cmd := exec.Command(os.Args[0], "1")
	cmd.Env = append(os.Environ(), "TEST_MAIN=echo")

	err := executor.ExecQuiet(cmd)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestExecQuietError(t *testing.T) {
	logger := log.New(bytes.NewBuffer([]byte{}), "", 0)
	executor := DefaultExecutor{logger}

	cmd := exec.Command(os.Args[0], "1")
	cmd.Env = append(os.Environ(), "TEST_MAIN=exit")

	err := executor.ExecQuiet(cmd)
	if err == nil {
		t.Errorf("expected error")
	}

	got := cmd.ProcessState.ExitCode()
	if got != 1 {
		fmt.Println(got)
		t.Errorf("expected %d, got %d", 1, got)
	}
}
