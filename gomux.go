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

type goMux struct {
	tmux     Tmux
	executor Executor
}

func (gomux goMux) execShellCommands(commands []string, path string) error {
	for _, c := range commands {
		cmd := exec.Command("/bin/sh", "-c", c)
		cmd.Dir = path

		_, err := gomux.executor.Exec(cmd)
		if err != nil {
			return err
		}
	}
	return nil
}

func (gomux goMux) setEnvVar(target string, env map[string]string) error {
	for key, value := range env {
		_, err := gomux.tmux.SetEnv(target, key, value)
		if err != nil {
			return err
		}
	}

	return nil
}

func (gomux goMux) switchOrAttach(target string, attach bool, insideTmuxSession bool) error {
	if insideTmuxSession && attach {
		return gomux.tmux.SwitchClient(target)
	} else if !insideTmuxSession {
		return gomux.tmux.Attach(target, os.Stdin, os.Stdout, os.Stderr)
	}
	return nil
}

func (gomux goMux) Stop(config Config, options Options, context Context) error {
	windows := options.Windows
	if len(windows) == 0 {
		sessionRoot := ExpandPath(config.Root)

		err := gomux.execShellCommands(config.Stop, sessionRoot)
		if err != nil {
			return err
		}
		_, err = gomux.tmux.StopSession(config.Session)
		return err
	}

	for _, w := range windows {
		err := gomux.tmux.KillWindow(config.Session + ":" + w)
		if err != nil {
			return err
		}
	}

	return nil
}

func (gomux goMux) Start(config Config, options Options, context Context) error {
	sessionName := config.Session + ":"
	sessionExists := gomux.tmux.SessionExists(sessionName)
	sessionRoot := ExpandPath(config.Root)

	windows := options.Windows
	attach := options.Attach

	rebalancePanesThreshold := config.RebalanceWindowsThreshold
	if rebalancePanesThreshold == 0 {
		rebalancePanesThreshold = defaultRebalancePanesThreshold
	}

	if !sessionExists {
		err := gomux.execShellCommands(config.BeforeStart, sessionRoot)
		if err != nil {
			return err
		}

		_, err = gomux.tmux.NewSession(config.Session, sessionRoot, defaultWindowName)
		if err != nil {
			return err
		}

		err = gomux.setEnvVar(config.Session, config.Env)
		if err != nil {
			return err
		}
	} else if len(windows) == 0 {
		return gomux.switchOrAttach(sessionName, attach, context.InsideTmuxSession)
	}

	for _, w := range config.Windows {
		if (len(windows) == 0 && w.Manual) || (len(windows) > 0 && !Contains(windows, w.Name)) {
			continue
		}

		windowRoot := ExpandPath(w.Root)
		if windowRoot == "" || !filepath.IsAbs(windowRoot) {
			windowRoot = filepath.Join(sessionRoot, w.Root)
		}

		window, err := gomux.tmux.NewWindow(sessionName, w.Name, windowRoot)
		if err != nil {
			return err
		}

		for _, c := range w.Commands {
			err := gomux.tmux.SendKeys(window, c)
			if err != nil {
				return err
			}
		}

		for pIndex, p := range w.Panes {
			paneRoot := ExpandPath(p.Root)
			if paneRoot == "" || !filepath.IsAbs(p.Root) {
				paneRoot = filepath.Join(windowRoot, p.Root)
			}

			newPane, err := gomux.tmux.SplitWindow(window, p.Type, paneRoot)

			if err != nil {
				return err
			}

			for _, c := range p.Commands {
				err = gomux.tmux.SendKeys(window+"."+newPane, c)
				if err != nil {
					return err
				}
			}

			if pIndex+1 >= rebalancePanesThreshold {
				_, err = gomux.tmux.SelectLayout(window, Tiled)
				if err != nil {
					return err
				}

			}
		}

		layout := w.Layout
		if layout == "" {
			layout = EvenHorizontal
		}

		_, err = gomux.tmux.SelectLayout(window, layout)
		if err != nil {
			return err
		}
	}

	err := gomux.tmux.KillWindow(sessionName + defaultWindowName)
	if err != nil {
		return err
	}

	err = gomux.tmux.RenumberWindows(sessionName)
	if err != nil {
		return err
	}

	if len(windows) == 0 && len(config.Windows) > 0 && !options.Detach {
		return gomux.switchOrAttach(sessionName+config.Windows[0].Name, attach, context.InsideTmuxSession)
	}

	return nil
}

func (gomux goMux) GetConfigFromSession(options Options, context Context) (Config, error) {
	config := Config{}

	tmuxSession, err := gomux.tmux.SessionName()
	if err != nil {
		return Config{}, err
	}
	config.Session = tmuxSession

	tmuxWindows, err := gomux.tmux.ListWindows(options.Project)
	if err != nil {
		return Config{}, err
	}

	for _, w := range tmuxWindows {
		tmuxPanes, err := gomux.tmux.ListPanes(options.Project + ":" + w.Id)
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
