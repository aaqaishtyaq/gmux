# gmux ![Gopher with tmux](https://github.com/dvit0/fsrv-cdn/raw/b7b656d9d06f4add2e4d72011e409472be40d826/files/go-tmux.png)

Tmux session manager.
Gmux automates your [tmux](https://github.com/tmux/tmux) workflow. You can create a single configuration file, and Gmux will create all the required windows and panes from it.

## Usage

```shell
gmux <command> [<project>] [-f, --file <file>] [-w, --windows <window>]... [-a, --attach] [-d, --debug]
```

### Options

```console
-f, --file A custom path to a config file
-w, --windows List of windows to start. If session exists, those windows will be attached to current session.
-a, --attach Force switch client for a session
-i, --inside-current-session Create all windows inside current session
-d, --debug Print all commands to ~/.config/gmux/gmux.log
--detach Detach session. The same as `-d` flag in the tmux
```

## Examples

## Getting started

To create a new project, or edit an existing one:

```shell
% gmux new work
% gmux edit work
```

To start/stop a project and all windows:

```shell
% gmux start work
% gmux stop work
```

### Example Config

Sample config should look like this.

```yaml
session: work

root: ~/Developer/work

before_start:
  - docker-compose -f work-backend/docker-compose.yml up -d # path relative to root

env:
  FOO: BAR

stop:
  - docker stop $(docker ps -q)

windows:
  - name: code
    root: work # a relative path to root
    manual: true # you can start this window only manually, using the -w arg
    layout: main-vertical
    commands:
      - docker-compose start
    panes:
      - type: horizontal
        root: .
        commands:
          - vim work-backend

  - name: infrastructure
    root: ~/Developer/work/work-backend
    layout: tiled
    panes:
      - type: horizontal
        root: .
        commands:
          - docker-compose up -d
          - docker-compose exec rails /bin/bash
          - clear
```
