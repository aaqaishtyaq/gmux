package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const defaultWindowName = "hr_def"

// Very wisely picked default value,
// after which panes will be rebalanced for each `split-window`
// Helps with "no space for new pane" error
const defaultRebalancePanesThreshold = 5

func Contains(slice []string, s string) bool {
	for _, e := range slice {
		if e == s {
			return true
		}
	}

	return false
}

func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		userHome, err := os.UserHomeDir()
		if err != nil {
			return path
		}

		return strings.Replace(path, "~", userHome, 1)
	}

	return path
}

type HR struct {
	tmux     Tmux
	executor Executor
}

func (hr HR) execShellCommands(commands []string, path string) error {
	for _, c := range commands {
		cmd := exec.Command("/bin/sh", "-c", c)
		cmd.Dir = path

		_, err := hr.executor.Exec(cmd)
		if err != nil {
			return err
		}
	}
	return nil
}

func (hr HR) setEnvVar(target string, env map[string]string) error {
	for key, value := range env {
		_, err := hr.tmux.SetEnv(target, key, value)
		if err != nil {
			return err
		}
	}

	return nil
}

func (hr HR) switchOrAttach(target string, attach bool, insideTmuxSession bool) error {
	if insideTmuxSession && attach {
		return hr.tmux.SwitchClient(target)
	} else if !insideTmuxSession {
		return hr.tmux.Attach(target, os.Stdin, os.Stdout, os.Stderr)
	}
	return nil
}

func (hr HR) Stop(config Config, options Options, context Context) error {
	windows := options.Windows
	if len(windows) == 0 {
		sessionRoot := ExpandPath(config.Root)

		err := hr.execShellCommands(config.Stop, sessionRoot)
		if err != nil {
			return err
		}
		_, err = hr.tmux.StopSession(config.Session)
		return err
	}

	for _, w := range windows {
		err := hr.tmux.KillWindow(config.Session + ":" + w)
		if err != nil {
			return err
		}
	}

	return nil
}

func (hr HR) Start(config Config, options Options, context Context) error {
	sessionName := config.Session + ":"
	sessionExists := hr.tmux.SessionExists(sessionName)
	sessionRoot := ExpandPath(config.Root)

	windows := options.Windows
	attach := options.Attach

	rebalancePanesThreshold := config.RebalanceWindowsThreshold
	if rebalancePanesThreshold == 0 {
		rebalancePanesThreshold = defaultRebalancePanesThreshold
	}

	if !sessionExists {
		err := hr.execShellCommands(config.BeforeStart, sessionRoot)
		if err != nil {
			return err
		}

		_, err = hr.tmux.NewSession(config.Session, sessionRoot, defaultWindowName)
		if err != nil {
			return err
		}

		err = hr.setEnvVar(config.Session, config.Env)
		if err != nil {
			return err
		}
	} else if len(windows) == 0 {
		return hr.switchOrAttach(sessionName, attach, context.InsideTmuxSession)
	}

	for _, w := range config.Windows {
		if (len(windows) == 0 && w.Manual) || (len(windows) > 0 && !Contains(windows, w.Name)) {
			continue
		}

		windowRoot := ExpandPath(w.Root)
		if windowRoot == "" || !filepath.IsAbs(windowRoot) {
			windowRoot = filepath.Join(sessionRoot, w.Root)
		}

		window, err := hr.tmux.NewWindow(sessionName, w.Name, windowRoot)
		if err != nil {
			return err
		}

		for _, c := range w.Commands {
			err := hr.tmux.SendKeys(window, c)
			if err != nil {
				return err
			}
		}

		for pIndex, p := range w.Panes {
			paneRoot := ExpandPath(p.Root)
			if paneRoot == "" || !filepath.IsAbs(p.Root) {
				paneRoot = filepath.Join(windowRoot, p.Root)
			}

			newPane, err := hr.tmux.SplitWindow(window, p.Type, paneRoot)

			if err != nil {
				return err
			}

			for _, c := range p.Commands {
				err = hr.tmux.SendKeys(window+"."+newPane, c)
				if err != nil {
					return err
				}
			}

			if pIndex+1 >= rebalancePanesThreshold {
				_, err = hr.tmux.SelectLayout(window, Tiled)
				if err != nil {
					return err
				}

			}
		}

		layout := w.Layout
		if layout == "" {
			layout = EvenHorizontal
		}

		_, err = hr.tmux.SelectLayout(window, layout)
		if err != nil {
			return err
		}
	}

	hr.tmux.KillWindow(sessionName + defaultWindowName)
	hr.tmux.RenumberWindows(sessionName)

	if len(windows) == 0 && len(config.Windows) > 0 && !options.Detach {
		return hr.switchOrAttach(sessionName+config.Windows[0].Name, attach, context.InsideTmuxSession)
	}

	return nil
}

func (hr HR) GetConfigFromSession(options Options, context Context) (Config, error) {
	config := Config{}

	tmuxSession, err := hr.tmux.SessionName()
	if err != nil {
		return Config{}, err
	}
	config.Session = tmuxSession

	tmuxWindows, err := hr.tmux.ListWindows(options.Project)
	if err != nil {
		return Config{}, err
	}

	for _, w := range tmuxWindows {
		tmuxPanes, err := hr.tmux.ListPanes(options.Project + ":" + w.Id)
		if err != nil {
			return Config{}, err
		}

		panes := []Pane{}
		for _, p := range tmuxPanes {
			root := p.Root
			if root == w.Root {
				root = ""
			}
			panes = append(panes, Pane{
				Root: root,
			})
		}

		config.Windows = append(config.Windows, Window{
			Name:   w.Name,
			Layout: w.Layout,
			Root:   w.Root,
			Panes:  panes,
		})
	}

	return config, nil
}
