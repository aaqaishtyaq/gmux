package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const defaultWindowName = "gomux_def"

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

type Gmux struct {
	tmux     Tmux
	executor Executor
}

func (gmux Gmux) execShellCommands(commands []string, path string) error {
	for _, c := range commands {
		cmd := exec.Command("/bin/sh", "-c", c)
		cmd.Dir = path

		_, err := gmux.executor.Exec(cmd)
		if err != nil {
			return err
		}
	}
	return nil
}

func (gmux Gmux) setEnvVariables(target string, env map[string]string) error {
	for key, value := range env {
		_, err := gmux.tmux.SetEnv(target, key, value)
		if err != nil {
			return err
		}
	}

	return nil
}

func (gmux Gmux) switchOrAttach(target string, attach bool, insideTmuxSession bool) error {
	if insideTmuxSession && attach {
		return gmux.tmux.SwitchClient(target)
	} else if !insideTmuxSession {
		return gmux.tmux.Attach(target, os.Stdin, os.Stdout, os.Stderr)
	}

	return nil
}

func (gmux Gmux) Stop(config Config, options Options, context Context) error {
	windows := options.Windows
	if len(windows) == 0 {
		sessionRoot := ExpandPath(config.Root)

		err := gmux.execShellCommands(config.Stop, sessionRoot)
		if err != nil {
			return err
		}
		_, err = gmux.tmux.StopSession(config.Session)
		return err
	}

	for _, w := range windows {
		err := gmux.tmux.KillWindow(config.Session + ":" + w)
		if err != nil {
			return err
		}
	}

	return nil
}

func (gmux Gmux) Start(config Config, options Options, context Context) error {
	sessionName := config.Session + ":"
	sessionExists := gmux.tmux.SessionExists(sessionName)
	sessionRoot := ExpandPath(config.Root)

	windows := options.Windows
	attach := options.Attach

	rebalancePanesThreshold := config.RebalanceWindowsThreshold
	if rebalancePanesThreshold == 0 {
		rebalancePanesThreshold = defaultRebalancePanesThreshold
	}

	if !sessionExists {
		err := gmux.execShellCommands(config.BeforeStart, sessionRoot)
		if err != nil {
			return err
		}

		_, err = gmux.tmux.NewSession(config.Session, sessionRoot, defaultWindowName)
		if err != nil {
			return err
		}

		err = gmux.setEnvVariables(config.Session, config.Env)
		if err != nil {
			return err
		}
	} else if len(windows) == 0 && !options.InsideCurrentSession {
		return gmux.switchOrAttach(sessionName, attach, context.InsideTmuxSession)
	}

	for _, w := range config.Windows {
		if (len(windows) == 0 && w.Manual) || (len(windows) > 0 && !Contains(windows, w.Name)) {
			continue
		}

		windowRoot := ExpandPath(w.Root)
		if windowRoot == "" || !filepath.IsAbs(windowRoot) {
			windowRoot = filepath.Join(sessionRoot, w.Root)
		}

		window, err := gmux.tmux.NewWindow(sessionName, w.Name, windowRoot)
		if err != nil {
			return err
		}

		for _, c := range w.Commands {
			err := gmux.tmux.SendKeys(window, c)
			if err != nil {
				return err
			}
		}

		for pIndex, p := range w.Panes {
			paneRoot := ExpandPath(p.Root)
			if paneRoot == "" || !filepath.IsAbs(p.Root) {
				paneRoot = filepath.Join(windowRoot, p.Root)
			}

			newPane, err := gmux.tmux.SplitWindow(window, p.Type, paneRoot)

			if err != nil {
				return err
			}

			for _, c := range p.Commands {
				err = gmux.tmux.SendKeys(window+"."+newPane, c)
				if err != nil {
					return err
				}
			}

			if pIndex+1 >= rebalancePanesThreshold {
				_, err = gmux.tmux.SelectLayout(window, Tiled)
				if err != nil {
					return err
				}

			}
		}

		layout := w.Layout
		switch layout {
		case EvenHorizontal, EvenVertical, MainHorizontal, MainVertical:
		default:
			layout = EvenHorizontal
		}

		_, err = gmux.tmux.SelectLayout(window, layout)
		if err != nil {
			return err
		}
	}

	if !options.InsideCurrentSession {
		err := gmux.tmux.KillWindow(sessionName + defaultWindowName)
		if err != nil {
			return err
		}

		err = gmux.tmux.RenumberWindows(sessionName)
		if err != nil {
			return err
		}
	}

	if len(windows) == 0 && len(config.Windows) > 0 && !options.Detach {
		return gmux.switchOrAttach(sessionName+config.Windows[0].Name, attach, context.InsideTmuxSession)
	}

	return nil
}

func (gmux Gmux) GetConfigFromSession(options Options, context Context) (Config, error) {
	config := Config{}

	tmuxSession, err := gmux.tmux.SessionName()
	if err != nil {
		return Config{}, err
	}
	config.Session = tmuxSession

	tmuxWindows, err := gmux.tmux.ListWindows(options.Project)
	if err != nil {
		return Config{}, err
	}

	for _, w := range tmuxWindows {
		tmuxPanes, err := gmux.tmux.ListPanes(options.Project + ":" + w.Id)
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
