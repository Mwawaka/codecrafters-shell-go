package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

const (
	fdStdout int = 1
	fdStderr int = 2
	none     int = 3
)

const (
	TokenWord              = iota
	TokenRedirectOut       // '>'
	TokenRedirectOutAppend // '>>'
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
		fmt.Fprint(os.Stdout, "$ ")
		os.Stdout.Sync()
		// os.Stdout.Write([]byte("$ "))

		command, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintln(os.Stderr, "error reading input:", err)
			os.Exit(1)
		}

		parts := tokenizer(command[:len(command)-1])
		// redirectIndex := slices.Index(parts, ">")
		redirectIndex := -1
		redirectType := TokenWord
		isAppend := false

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

		for i, token := range parts {
			tt := tokenType(token)
			if tt == TokenRedirectOut || tt == TokenRedirectOutAppend {
				redirectIndex = i
				redirectType = tt
				isAppend = (tt == TokenRedirectOutAppend)
				break //TODO: remove in real-shell
			} else if token == ">>" {
				redirectIndex = i
				isAppend = true
				break
			}
		}
		fmt.Println("redirect type:", redirectType)
		if redirectIndex != -1 && redirectIndex+1 < len(parts) {
			args := parts[1:redirectIndex]
			filename := parts[redirectIndex+1]
			fileDescriptor := fdStdout

			if redirectIndex > 0 && parts[redirectIndex-1] == "2" {
				fileDescriptor = fdStderr
				args = parts[1 : redirectIndex-1]
			}

			err := handleRedirect(cmdName, filename, args, commands, fileDescriptor, isAppend)

			if err != nil {
				var exitErr *exec.ExitError

				if errors.As(err, &exitErr) {
					fmt.Print()
				} else if errors.Is(err, exec.ErrNotFound) {
					fmt.Fprintf(os.Stderr, "%s: command not found\n", cmdName)
				} else {
					fmt.Fprintln(os.Stderr, err)
				}
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

		if err := runExternal(cmdName, parts[1:], os.Stdout, none); err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				continue
			}
			fmt.Printf("%s: command not found\n", cmdName)
		}
	}
}

func tokenType(token string) int {
	switch token {
	case ">":
		return TokenRedirectOut
	case ">>":
		return TokenRedirectOutAppend
	default:
		return TokenWord
	}
}

func handleRedirect(cmdName, filename string, args []string, commands map[string]CommandHandler, fileDescriptor int, appendMode bool) error {
	var buffer bytes.Buffer

	if handler, exists := commands[cmdName]; exists {
		out, err := handler(args)

		if err != nil {
			return err
		}

		if fileDescriptor == fdStdout {
			return writeToFile(filename, []byte(out+"\n"), appendMode)
		}

		writeToFile(filename, []byte{}, appendMode)
		fmt.Println(out)
		return nil
	}

	if err := runExternal(cmdName, args, &buffer, fileDescriptor); err != nil {
		writeToFile(filename, buffer.Bytes(), appendMode)
		return err
	}

	return writeToFile(filename, buffer.Bytes(), appendMode)
}

func tokenizer(command string) []string {
	runes := []rune(command)
	var builder strings.Builder
	tokens := []string{}
	inSingleQuote := false
	inDoubleQuote := false
	inBackSlash := false

	for i := 0; i < len(runes); i++ {
		r := runes[i]
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
			nextRune, hasNext := peekNext(runes, i)

			if hasNext && nextRune == '>' {
				if builder.Len() > 0 {
					tokens = append(tokens, builder.String())
					builder.Reset()
				}
				tokens = append(tokens, ">>")
				i++
			} else {
				if builder.String() == "1" {
					builder.Reset()
				}

				if builder.Len() > 0 {
					tokens = append(tokens, builder.String())
					builder.Reset()
				}

				tokens = append(tokens, ">")
			}
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
	// fmt.Println("token length", len(tokens))
	return tokens
}

func peekNext(runes []rune, i int) (rune, bool) {
	nextIndx := i + 1
	if nextIndx < len(runes) {
		return runes[nextIndx], true
	}
	return 0, false
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

func runExternal(cmdName string, args []string, writer io.Writer, fileDescriptor int) error {

	if _, err := exec.LookPath(cmdName); err != nil {
		return err
	}

	cmd := exec.Command(cmdName, args...)

	switch fileDescriptor {
	case fdStdout:
		cmd.Stdout = writer
		cmd.Stderr = os.Stderr
	case fdStderr:
		cmd.Stdout = os.Stdout
		cmd.Stderr = writer
	default:
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

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

	if len(args) > 0 {
		if args[0] == "~" {
			return chDirToHome()
		} else {
			err := os.Chdir(args[0])
			if err != nil {
				return fmt.Errorf("cd: %s: No such file or directory", args[0])
			}
		}
	}

	if len(args) == 0 {
		path, err := pwd([]string{})
		if err != nil {
			return err
		}
		return os.Chdir(path)
	}

	return nil
}

func chDirToHome() error {
	home := os.Getenv("HOME")
	if home == "" {
		return fmt.Errorf("HOME is not set")
	}
	return os.Chdir(home)
}
