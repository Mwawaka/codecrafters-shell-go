package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"
)

type CommandHandler func(args []string) (string, error)

func main() {

	var commands = map[string]CommandHandler{
		"echo": echo,
		"pwd":  pwd,
	}

	commands["type"] = func(args []string) (string, error) {
		return typeCmd(commands, args)
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("$ ")
		command, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintln(os.Stderr, "error reading input:", err)
			os.Exit(1)
		}

		parts := tokenizer(command[:len(command)-1])
		redirectIndex := slices.Index(parts, ">")
		if len(parts) == 0 || parts[0] == "" {
			continue
		}

		cmdName := parts[0]

		if cmdName == "exit" {
			exit()
		}

		if cmdName == "cd" {
			if err := cd(parts[1:]); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}

			continue
		}

		if redirectIndex != -1 && redirectIndex+1 < len(parts) {
			var buffer bytes.Buffer
			args := parts[1:redirectIndex]
			filename := parts[redirectIndex+1]
			if handler, exists := commands[cmdName]; exists {
				out, err := handler(args)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)

				} else {

					if err = writeToFile(filename, []byte(out+"\n")); err != nil {
						fmt.Fprintln(os.Stderr, err)
					}
				}
				continue
			}

			if err := runExternal(cmdName, args, &buffer); err != nil {
				var exitErr *exec.ExitError

				if errors.As(err, &exitErr) {
					continue
				}

				fmt.Printf("%s: command not found\n", cmdName)
				continue
			}

			if err = writeToFile(filename, buffer.Bytes()); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
			continue
		}

		if handler, exists := commands[cmdName]; exists {
			out, err := handler(parts[1:])
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				continue
			}
			fmt.Println(out)
			continue
		}

		if err := runExternal(cmdName, parts[1:], os.Stdout); err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				continue
			}
			fmt.Printf("%s: command not found\n", cmdName)
		}
	}
}

func tokenizer(command string) []string {
	// TODO: handling backticks
	var builder strings.Builder
	tokens := []string{}

	inSingleQuote := false
	inDoubleQuote := false
	inBackSlash := false

	for _, r := range command {
		if inBackSlash {
			if inDoubleQuote && !isEscapableInDoubleQuote(r) {
				builder.WriteRune('\\')
			}

			builder.WriteRune(r)
			inBackSlash = false
			continue
		}

		if r == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
			continue
		}

		if r == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
			continue
		}

		if r == '\\' && !inSingleQuote {
			inBackSlash = true
			continue
		}

		if r == '>' && !inSingleQuote && !inDoubleQuote {
			if builder.String() == "1" {
				builder.Reset()
			}

			if builder.Len() > 0 {
				tokens = append(tokens, builder.String())
				builder.Reset()
			}

			tokens = append(tokens, ">")
			continue
		}

		if r == ' ' && !inSingleQuote && !inDoubleQuote {
			if builder.Len() > 0 {
				tokens = append(tokens, builder.String())
				builder.Reset()
			}

			continue
		}

		builder.WriteRune(r)
	}

	if builder.Len() > 0 {
		tokens = append(tokens, builder.String())
	}

	// fmt.Println("tokens:", tokens)
	return tokens
}

func isEscapableInDoubleQuote(r rune) bool {
	return r == '"' || r == '\\' || r == '$' || r == '`' || r == '\n'
}
func exit() {
	os.Exit(0)
}

func echo(args []string) (string, error) {
	return strings.Join(args, " "), nil
}

func typeCmd(commands map[string]CommandHandler, args []string) (string, error) {

	msg := make([]string, len(args))

	for i, arg := range args {
		_, exists := commands[arg]

		if arg == "exit" || arg == "cd" || exists {
			msg[i] = fmt.Sprintf("%s is a shell builtin", arg)
			continue
		}

		path, err := exec.LookPath(arg)

		if err != nil {
			msg[i] = fmt.Sprintf("%s: not found", arg)
			continue
		}

		msg[i] = fmt.Sprintf("%s is %s", arg, path)
	}
	return strings.Join(msg, "\n"), nil
}

func runExternal(file string, args []string, writer io.Writer) error {
	if _, err := exec.LookPath(file); err != nil {
		return err
	}

	cmd := exec.Command(file, args...)
	cmd.Stdout = writer
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func pwd(args []string) (string, error) {
	path, err := os.Getwd()

	if err != nil {
		return "", err
	}

	return path, nil
}

func cd(args []string) error {

	if len(args) > 1 {
		return fmt.Errorf("cd: too many arguments")
	}

	if len(args) == 0 || args[0] == "~" {
		return chDirToHome()
	}

	if err := os.Chdir(args[0]); err != nil {
		return fmt.Errorf("cd: %s: No such file or directory", args[0])

	}
	return nil
}

func chDirToHome() error {
	return os.Chdir(os.Getenv("HOME"))
}
