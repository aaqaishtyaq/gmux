package main

import (
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

var testTable = map[string]struct {
	config           Config
	options          Options
	context          Context
	startCommands    []string
	stopCommands     []string
	commanderOutputs []string
}{
	"test with 1 window": {
		Config{
			Session:     "test-session",
			Root:        "~/root",
			BeforeStart: []string{"command1", "command2"},
			Windows: []Window{
				{
					Name:     "win1",
					Commands: []string{"command1"},
				},
			},
		},
		Options{},
		Context{},
		[]string{
			"tmux has-session -t test-session:",
			"/bin/sh -c command1",
			"/bin/sh -c command2",
			"tmux new -Pd -s test-session -n gomux_def -c gmux/root",
			"tmux neww -Pd -t test-session: -c gmux/root -F #{window_id} -n win1",
			"tmux send-keys -t win1 command1 Enter",
			"tmux select-layout -t win1 even-horizontal",
			"tmux kill-window -t test-session:gomux_def",
			"tmux move-window -r -s test-session: -t test-session:",
			"tmux attach -d -t test-session:win1",
		},
		[]string{
			"tmux kill-session -t test-session",
		},
		[]string{"test-session", "win1"},
	},
	"test with 1 window and Detach: true": {
		Config{
			Session:     "test-session",
			Root:        "root",
			BeforeStart: []string{"command1", "command2"},
			Windows: []Window{
				{
					Name: "win1",
				},
			},
		},
		Options{
			Detach: true,
		},
		Context{},
		[]string{
			"tmux has-session -t test-session:",
			"/bin/sh -c command1",
			"/bin/sh -c command2",
			"tmux new -Pd -s test-session -n gomux_def -c root",
			"tmux neww -Pd -t test-session: -c root -F #{window_id} -n win1",
			"tmux select-layout -t xyz even-horizontal",
			"tmux kill-window -t test-session:gomux_def",
			"tmux move-window -r -s test-session: -t test-session:",
		},
		[]string{
			"tmux kill-session -t test-session",
		},
		[]string{"xyz"},
	},
	"test with multiple windows and panes": {
		Config{
			Session: "test-session",
			Root:    "root",
			Windows: []Window{
				{
					Name:   "win1",
					Manual: false,
					Layout: "main-horizontal",
					Panes: []Pane{
						{
							Type:     "horizontal",
							Commands: []string{"command1"},
						},
					},
				},
				{
					Name:   "win2",
					Manual: true,
					Layout: "tiled",
				},
			},
			Stop: []string{
				"stop1",
				"stop2 -d --foo=bar",
			},
		},
		Options{},
		Context{},
		[]string{
			"tmux has-session -t test-session:",
			"tmux new -Pd -s test-session -n gomux_def -c root",
			"tmux neww -Pd -t test-session: -c root -F #{window_id} -n win1",
			"tmux split-window -Pd -h -t win1 -c root -F #{pane_id}",
			"tmux send-keys -t win1.1 command1 Enter",
			"tmux select-layout -t win1 main-horizontal",
			"tmux kill-window -t test-session:gomux_def",
			"tmux move-window -r -s test-session: -t test-session:",
			"tmux attach -d -t test-session:win1",
		},
		[]string{
			"/bin/sh -c stop1",
			"/bin/sh -c stop2 -d --foo=bar",
			"tmux kill-session -t test-session",
		},
		[]string{"test-session", "test-session", "win1", "1"},
	},
	"test start windows from option's Windows parameter": {
		Config{
			Session: "test-session",
			Root:    "root",
			Windows: []Window{
				{
					Name:   "win1",
					Manual: false,
				},
				{
					Name:   "win2",
					Manual: true,
				},
			},
		},
		Options{
			Windows: []string{"win2"},
		},
		Context{},
		[]string{
			"tmux has-session -t test-session:",
			"tmux new -Pd -s test-session -n gomux_def -c root",
			"tmux neww -Pd -t test-session: -c root -F #{window_id} -n win2",
			"tmux select-layout -t xyz even-horizontal",
			"tmux kill-window -t test-session:gomux_def",
			"tmux move-window -r -s test-session: -t test-session:",
		},
		[]string{
			"tmux kill-window -t test-session:win2",
		},
		[]string{"xyz"},
	},
	"test attach to the existing session": {
		Config{
			Session: "test-session",
			Root:    "root",
			Windows: []Window{
				{Name: "win1"},
			},
		},
		Options{},
		Context{},
		[]string{
			"tmux has-session -t test-session:",
			"tmux attach -d -t test-session:",
		},
		[]string{
			"tmux kill-session -t test-session",
		},
		[]string{""},
	},
	"test start a new session from another tmux session": {
		Config{
			Session: "test-session",
			Root:    "root",
		},
		Options{Attach: false},
		Context{InsideTmuxSession: true},
		[]string{
			"tmux has-session -t test-session:",
			"tmux new -Pd -s test-session -n gomux_def -c root",
			"tmux kill-window -t test-session:gomux_def",
			"tmux move-window -r -s test-session: -t test-session:",
		},
		[]string{
			"tmux kill-session -t test-session",
		},
		[]string{"xyz"},
	},
	"test switch a client from another tmux session": {
		Config{
			Session: "test-session",
			Root:    "root",
			Windows: []Window{
				{Name: "win1"},
			},
		},
		Options{Attach: true},
		Context{InsideTmuxSession: true},
		[]string{
			"tmux has-session -t test-session:",
			"tmux switch-client -t test-session:",
		},
		[]string{
			"tmux kill-session -t test-session",
		},
		[]string{""},
	},
	"test create new windows in current session": {
		Config{
			Session: "test-session",
			Root:    "root",
			Windows: []Window{
				{Name: "win1"},
			},
		},
		Options{
			InsideCurrentSession: true,
		},
		Context{InsideTmuxSession: true},
		[]string{
			"tmux has-session -t test-session:",
			"tmux neww -Pd -t test-session: -c root -F #{window_id} -n win1",
			"tmux select-layout -t  even-horizontal",
		},
		[]string{
			"tmux kill-session -t test-session",
		},
		[]string{""},
	},
}

type MockExecutor struct {
	Commands []string
	Outputs  []string
}

func (c *MockExecutor) Exec(cmd *exec.Cmd) (string, error) {
	c.Commands = append(c.Commands, strings.Join(cmd.Args, " "))

	output := ""
	if len(c.Outputs) > 1 {
		output, c.Outputs = c.Outputs[0], c.Outputs[1:]
	} else if len(c.Outputs) == 1 {
		output = c.Outputs[0]
	}

	return output, nil
}

func (c *MockExecutor) ExecQuiet(cmd *exec.Cmd) error {
	c.Commands = append(c.Commands, strings.Join(cmd.Args, " "))
	return nil
}

func TestStartStopSession(t *testing.T) {
	os.Setenv("HOME", "gmux") // Needed for testing ExpandPath function

	for testDescription, params := range testTable {

		t.Run("start session: "+testDescription, func(t *testing.T) {
			executor := &MockExecutor{[]string{}, params.commanderOutputs}
			tmux := Tmux{executor}
			gmux := Gmux{tmux, executor}

			err := gmux.Start(params.config, params.options, params.context)
			if err != nil {
				t.Fatalf("error %v", err)
			}

			if !reflect.DeepEqual(params.startCommands, executor.Commands) {
				t.Errorf("expected\n%s\ngot\n%s", strings.Join(params.startCommands, "\n"), strings.Join(executor.Commands, "\n"))
			}
		})

		t.Run("stop session: "+testDescription, func(t *testing.T) {
			executor := &MockExecutor{[]string{}, params.commanderOutputs}
			tmux := Tmux{executor}
			gmux := Gmux{tmux, executor}

			err := gmux.Stop(params.config, params.options, params.context)
			if err != nil {
				t.Fatalf("error %v", err)
			}

			if !reflect.DeepEqual(params.stopCommands, executor.Commands) {
				t.Errorf("expected\n%s\ngot\n%s", strings.Join(params.stopCommands, "\n"), strings.Join(executor.Commands, "\n"))
			}
		})

	}
}

func TestPrintCurrentSession(t *testing.T) {
	expectedConfig := Config{
		Session: "session_name",
		Windows: []Window{
			{
				Name:   "win1",
				Root:   "root",
				Layout: "layout",
				Panes: []Pane{
					{},
					{
						Root: "/tmp",
					},
				},
			},
		},
	}

	executor := &MockExecutor{[]string{}, []string{
		"session_name",
		"id1;win1;layout;root",
		"root\n/tmp",
	}}
	tmux := Tmux{executor}

	gmux := Gmux{tmux, executor}

	actualConfig, err := gmux.GetConfigFromSession(Options{Project: "test"}, Context{})
	if err != nil {
		t.Fatalf("error %v", err)
	}

	if !reflect.DeepEqual(expectedConfig, actualConfig) {
		t.Errorf("expected %v, got %v", expectedConfig, actualConfig)
	}
}
