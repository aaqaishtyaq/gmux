package main

import (
	"errors"
	"strings"

	"github.com/spf13/pflag"
)

const (
	CommandStart = "start"
	CommandStop  = "stop"
	CommandNew   = "new"
	CommandEdit  = "edit"
	CommandList  = "list"
	CommandPrint = "print"
)

var validCommands = []string{CommandStart, CommandStop, CommandNew, CommandEdit, CommandList, CommandPrint}

type Options struct {
	Command  string
	Project  string
	Config   string
	Windows  []string
	Settings map[string]string
	Attach   bool
	Detach   bool
	Debug    bool
}

var ErrHelp = errors.New("help requested")

const (
	WindowsUsage = "List of windows to start. If session exists, those windows will be attached to current session."
	AttachUsage  = "Force switch client for a session"
	DetachUsage  = "Detach tmux session. The same as -d flag in the tmux"
	DebugUsage   = "Print all commands to ~/.config/gomux/gomux.log"
	FileUsage    = "A custom path to a config file"
)

// Creates a new FlagSet.
// Moved it to a variable to be able to override it in the tests.
var NewFlagSet = func(cmd string) *pflag.FlagSet {
	f := pflag.NewFlagSet(cmd, pflag.ContinueOnError)
	return f
}

func ParseOptions(argv []string, helpRequested func()) (Options, error) {
	if len(argv) == 0 {
		helpRequested()
		return Options{}, ErrHelp
	}

	if argv[0] == "--help" || argv[0] == "-h" {
		helpRequested()
		return Options{}, ErrHelp
	}

	cmd := argv[0]
	if !Contains(validCommands, cmd) {
		helpRequested()
		return Options{}, ErrHelp
	}

	flags := NewFlagSet(cmd)

	config := flags.StringP("file", "f", "", FileUsage)
	windows := flags.StringArrayP("windows", "w", []string{}, WindowsUsage)
	attach := flags.BoolP("attach", "a", false, AttachUsage)
	detach := flags.Bool("detach", false, DetachUsage)
	debug := flags.BoolP("debug", "d", false, DebugUsage)

	err := flags.Parse(argv)

	if err == pflag.ErrHelp {
		return Options{}, ErrHelp
	}

	if err != nil {
		return Options{}, err
	}

	var project string
	if *config == "" && len(argv) > 1 {
		project = argv[1]
	}

	if strings.Contains(project, ":") {
		parts := strings.Split(project, ":")
		project = parts[0]
		wl := strings.Split(parts[1], ",")
		windows = &wl
	}

	settings := make(map[string]string)
	userSettings := flags.Args()[1:]
	if len(userSettings) > 0 {
		for _, kv := range userSettings {
			s := strings.Split(kv, "=")
			if len(s) < 2 {
				continue
			}
			settings[s[0]] = s[1]
		}
	}

	return Options{
		Project:  project,
		Config:   *config,
		Command:  cmd,
		Settings: settings,
		Windows:  *windows,
		Attach:   *attach,
		Detach:   *detach,
		Debug:    *debug,
	}, nil
}
