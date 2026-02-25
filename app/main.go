package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/chzyer/readline"
	"github.com/codecrafters-io/shell-starter-go/internal/autocompleter"
	"github.com/codecrafters-io/shell-starter-go/internal/builtins"
	"github.com/codecrafters-io/shell-starter-go/internal/executor"
	"github.com/codecrafters-io/shell-starter-go/internal/parser"
	"github.com/codecrafters-io/shell-starter-go/internal/redirect"
)

func main() {
	var history = builtins.NewHistory()

	var commands = map[string]builtins.CommandHandler{
		"echo":    builtins.Echo,
		"pwd":     builtins.Pwd,
		"history": history.Display,
	}

	commands["type"] = func(args []string) (string, error) {
		return builtins.Type(commands, args)
	}

	completer := autocompleter.NewTabCompleter([]string{
		"pwd", "echo", "exit", "type", "cd", "history",
	})

	reader, err := readline.NewEx(&readline.Config{
		Prompt:       "$ ",
		AutoComplete: completer,
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "error starting shell: %v", err)
		os.Exit(1)
	}

	defer reader.Close()

	for {
		command, err := reader.Readline()

		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Fprintln(os.Stderr, "error reading input:", err)
			continue
		}

		trimmedInput := strings.TrimSpace(command)

		if trimmedInput != "" {
			history.Add(trimmedInput)
		}

		parts, err := parser.Parse(trimmedInput)

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}

		if len(parts) == 0 || parts[0] == "" {
			continue
		}

		pipeline := redirect.Pipe(parts)

		if len(pipeline) == 0 {
			continue
		}

		cmdName := pipeline[0][0]

		if cmdName == "exit" {
			break
		}

		if cmdName == "cd" {
			if err := builtins.Cd(parts[1:]); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}

			continue
		}

		err = executor.Execute(pipeline, commands)

		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {

			} else if errors.Is(err, exec.ErrNotFound) {
				fmt.Fprintf(os.Stderr, "%s: command not found\n", cmdName)
			} else {
				fmt.Fprintln(os.Stderr, err)
			}
		}
	}
}
