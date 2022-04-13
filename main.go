package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/aaqaishtyaq/gmux/config"
	"github.com/aaqaishtyaq/gmux/executor"
	"github.com/aaqaishtyaq/gmux/tmux"

	"gopkg.in/yaml.v2"
)

var version = "1.2.0"

var usage = fmt.Sprintf(`gmux - session manager for tmux. Version %s

Usage:
	gmux <command> [<project>] [-f, --file <file>] [-w, --windows <window>]... [-a, --attach]
	[-d, --debug] [--detach] [-i, --inside-current-session] [<key>=<value>]...

Options:
	-f, --file %s
	-w, --windows %s
	-a, --attach %s
	-i, --inside-current-session %s
	-d, --debug %s
	--detach %s

Commands:
	list    list available project configurations
	edit    edit project configuration
	new     new project configuration
	start   start project session
	stop    stop project session
	print   session configuration to stdout

	Examples:
	$ gmux list
	$ gmux edit work
	$ gmux new work
	$ gmux start work
	$ gmux start work:win1
	$ gmux start work -w win1
	$ gmux start work:win1,win2
	$ gmux stop work
	$ gmux start work --attach
	$ gmux print > ~/.config/gmux/work.yml
`, version, FileUsage, WindowsUsage, AttachUsage, InsideCurrentSessionUsage, DebugUsage, DetachUsage)

func main() {
	options, err := ParseOptions(os.Args[1:], func() {
		fmt.Fprintln(os.Stdout, usage)
		os.Exit(0)
	})

	if err == ErrHelp {
		os.Exit(0)
	}

	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"Cannot parse command line options: %q",
			err.Error(),
		)
		os.Exit(1)
	}

	userConfigDir := filepath.Join(ExpandPath("~/"), ".config/gmux")

	var configPath string
	if options.Config != "" {
		configPath = options.Config
	} else {
		configPath = filepath.Join(userConfigDir, options.Project+".yaml")
	}

	var logger *log.Logger
	if options.Debug {
		logFile, err := os.Create(filepath.Join(userConfigDir, "gmux.log"))
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
		}
		logger = log.New(logFile, "", 0)
	}

	executor := executor.DefaultExecutor{Logger: logger}
	tmux := tmux.Tmux{Executor: executor}
	gmux := Gmux{tmux, executor}
	context := CreateContext()

	switch options.Command {
	case CommandStart:
		if len(options.Windows) == 0 {
			fmt.Println("Starting a new session...")
		} else {
			fmt.Println("Starting new windows...")
		}
		conf, err := config.GetConfig(configPath, options.Settings)
		if err != nil {
			fmt.Fprint(os.Stderr, err.Error())
			os.Exit(1)
		}

		err = gmux.Start(conf, options, context)
		if err != nil {
			fmt.Println("Oops, an error occurred! Rolling back...")
			fmt.Fprint(os.Stderr, err.Error())
			_ = gmux.Stop(conf, options, context)
			os.Exit(1)
		}

	case CommandStop:
		if len(options.Windows) == 0 {
			fmt.Println("Terminating session...")
		} else {
			fmt.Println("Killing windows...")
		}
		conf, err := config.GetConfig(configPath, options.Settings)
		if err != nil {
			fmt.Fprint(os.Stderr, err.Error())
			os.Exit(1)
		}

		err = gmux.Stop(conf, options, context)
		if err != nil {
			fmt.Fprint(os.Stderr, err.Error())
			os.Exit(1)
		}

	case CommandNew, CommandEdit:
		err := config.EditConfig(configPath)
		if err != nil {
			fmt.Fprint(os.Stderr, err.Error())
			os.Exit(1)
		}
	case CommandList:
		configs, err := config.ListConfigs(userConfigDir)
		if err != nil {
			fmt.Fprint(os.Stderr, err.Error())
			os.Exit(1)
		}

		fmt.Println(strings.Join(configs, "\n"))
	case CommandPrint:
		conf, err := gmux.GetConfigFromSession(options, context)
		if err != nil {
			fmt.Fprint(os.Stderr, err.Error())
			os.Exit(1)
		}

		d, err := yaml.Marshal(&conf)
		if err != nil {
			fmt.Fprint(os.Stderr, err.Error())
			os.Exit(1)
		}

		fmt.Println(string(d))
	}
}
