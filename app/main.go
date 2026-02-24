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

	var commands = map[string]builtins.CommandHandler{
		"echo": builtins.Echo,
		"pwd":  builtins.Pwd,
	}

	commands["type"] = func(args []string) (string, error) {
		return builtins.Type(commands, args)
	}

	completer := autocompleter.NewTabCompleter([]string{
		"pwd", "echo", "exit", "type",
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

		parts, err := parser.Parse(strings.TrimSpace(command))

		if err!=nil{
			fmt.Fprintln(os.Stderr,err)
			continue
		}

		if len(parts) == 0 || parts[0] == "" {
			continue
		}

		pipeline:=redirect.Pipe(parts)

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

		var filename string
		args := pipeline[0][1:]
		redirectIndex := -1
		appendMode := false
		fileDescriptor := redirect.FdStdout

		for i, token := range parts {
			fd, isAppend, isRedirect := redirect.Redirect(token)

			if isRedirect {
				redirectIndex = i
				fileDescriptor = fd
				appendMode = isAppend
				break
			}
		}

		if redirectIndex != -1 && redirectIndex+1 < len(parts) {
			filename = parts[redirectIndex+1]
			args = parts[1:redirectIndex]
		}

		err = executor.Execute(cmdName, filename, args, commands, fileDescriptor, appendMode)

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
