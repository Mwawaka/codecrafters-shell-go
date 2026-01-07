package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/chzyer/readline"
)

const (
	fdStdin  int = 0
	fdStdout int = 1
	fdStderr int = 2
)

var (
	tabCount     int
	lastInput    string
	builtinNames = []string{
		"ped", "echo", "exit", "type",
	}
)

type CommandHandler func(args []string) (string, error)

type TabCompleter struct{}

func (t *TabCompleter) Do(line []rune, pos int) (newLine [][]rune, length int) {
	currentInput := string(line[:pos])

	// Reset count if input changed
	if currentInput != lastInput {
		tabCount = 0
		lastInput = currentInput
	}

	tabCount++

	if tabCount == 1 {
		// Check builtins first
		builtinMatches := listCommands(currentInput)

		if len(builtinMatches) == 1 {
			// Autocomplete single builtin
			completion := builtinMatches[0] + " "
			return [][]rune{[]rune(completion[len(currentInput):])}, len(currentInput)
		}

		// No single builtin match: ring bell
		os.Stdout.Write([]byte("\x07"))
		os.Stdout.Sync()
		return [][]rune{}, len(currentInput)
	}

	if tabCount == 2 {
		// Show executable matches
		matches := listExecutables(currentInput)

		if len(matches) > 0 {
			sort.Strings(matches)
			fmt.Fprintf(os.Stdout, "\n%s\n", strings.Join(matches, "  "))

		}

		tabCount = 0
		return [][]rune{[]rune("")}, len(currentInput)
	}

	return nil, len(currentInput)
}

func main() {

	var builtins = map[string]CommandHandler{
		"echo": echo,
		"pwd":  pwd,
	}

	builtins["type"] = func(args []string) (string, error) {
		return typeCmd(builtins, args)
	}

	completer := &TabCompleter{}

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

		parts := parse(strings.TrimSpace(command))

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

		var filename string
		args := parts[1:]
		redirectIndex := -1
		appendMode := false
		fileDescriptor := fdStdout

		for i, token := range parts {
			fd, isAppend, isRedirect := parseRedirect(token)

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

		err = execute(cmdName, filename, args, builtins, fileDescriptor, appendMode)

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

func listCommands(prefix string) []string {
	var matches []string

	for _, cmd := range builtinNames {
		if strings.HasPrefix(cmd, prefix) {
			matches = append(matches, cmd)
		}
	}

	beep(matches)
	return matches
}

func listExecutables(prefix string) []string {
	var matches []string
	seen := make(map[string]bool)
	pathEnv := os.Getenv("PATH")

	if pathEnv == "" {
		return nil
	}

	separator := ":" //Unix based systems

	if runtime.GOOS == "windows" {
		separator = ";"
	}

	dirs := strings.Split(pathEnv, separator)

	for _, dir := range dirs {
		if dir == "" {
			continue
		}

		entries, err := os.ReadDir(dir)

		if err != nil {
			continue // skips bad dirs
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue // skips subdirs
			}

			name := entry.Name()

			// no auto completion for hidden files by default
			if strings.HasPrefix(name, ".") {
				continue
			}

			// check executable (Unix only - Windows alll files are "executable" )
			if runtime.GOOS != "windows" {
				info, err := entry.Info()

				if err != nil {
					continue
				}

				// check if file is executable
				if info.Mode()&0111 == 0 {
					continue
				}
			}

			if strings.HasPrefix(name, prefix) && !seen[name] {
				matches = append(matches, name)
				seen[name] = true
			}
		}
	}

	beep(matches)
	return matches
}

func beep(matches []string) {
	if len(matches) == 0 {
		os.Stdout.Write([]byte("\x07"))
		os.Stdout.Sync()
	}
}
func execute(command, filename string, args []string, builtins map[string]CommandHandler, fileDescriptor int, appendMode bool) error {
	var buffer bytes.Buffer
	var stdout io.Writer = os.Stdout
	var stderr io.Writer = os.Stderr

	if handler, exists := builtins[command]; exists {
		out, err := handler(args)

		if err != nil {
			return err
		}

		if filename != "" {
			data := []byte(out + "\n")

			if fileDescriptor == fdStderr {
				data = []byte{}
				os.Stdout.WriteString(out + "\n")
			}
			return writeToFile(filename, data, appendMode)
		}

		fmt.Fprintln(os.Stdout, out)

		return nil
	}

	if filename != "" {
		switch fileDescriptor {
		case fdStdout:
			stdout = &buffer
		case fdStderr:
			stderr = &buffer
		}
	}

	err := runExternal(command, args, stdout, stderr)

	if filename != "" {
		return writeToFile(filename, buffer.Bytes(), appendMode)
	}

	return err
}

func parse(command string) []string {
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
			currentToken := builder.String()

			if len(currentToken) == 1 && currentToken[0] >= '0' && currentToken[0] <= '9' {
				builder.Reset()

				if hasNext && nextRune == '>' {
					tokens = append(tokens, currentToken+">>")
					i++
				} else {
					tokens = append(tokens, currentToken+">")
				}
				continue
			}

			if hasNext && nextRune == '>' {
				flush(&builder, &tokens)
				tokens = append(tokens, ">>")
				i++
				continue
			}

			flush(&builder, &tokens)
			tokens = append(tokens, ">")
			builder.Reset()
			continue
		}

		if unicode.IsSpace(r) && !inSingleQuote && !inDoubleQuote {
			flush(&builder, &tokens)
			continue
		}

		builder.WriteRune(r)
	}

	flush(&builder, &tokens)
	// fmt.Println("Tokens: ", tokens)
	return tokens
}

func peekNext(runes []rune, i int) (rune, bool) {
	nextIndx := i + 1
	if nextIndx < len(runes) {
		return runes[nextIndx], true
	}
	return 0, false
}

func flush(b *strings.Builder, tokens *[]string) {
	if b.Len() > 0 {
		*tokens = append(*tokens, b.String())
		b.Reset()
	}
}

func isEscapableInDoubleQuote(r rune) bool {
	return r == '"' || r == '\\' || r == '$' || r == '`' || r == '\n'
}

func parseRedirect(token string) (fd int, isAppend, isRedirect bool) {
	var prefix string

	if strings.HasSuffix(token, ">>") {
		isRedirect = true
		isAppend = true
		prefix = strings.TrimSuffix(token, ">>")
	} else if strings.HasSuffix(token, ">") {
		isRedirect = true
		isAppend = false
		prefix = strings.TrimSuffix(token, ">")
	} else {
		return 0, false, false
	}

	if prefix == "" {
		return 1, isAppend, true
	}

	fd, err := strconv.Atoi(prefix)

	if err != nil {
		return 0, false, false
	}

	return fd, isAppend, isRedirect
}

func exit() {
	os.Exit(0)
}

func echo(args []string) (string, error) {
	return strings.Join(args, " "), nil
}

func typeCmd(builtins map[string]CommandHandler, args []string) (string, error) {

	msg := make([]string, len(args))

	for i, arg := range args {
		_, exists := builtins[arg]

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

func runExternal(cmdName string, args []string, stdout, stderr io.Writer) error {
	cmd := exec.Command(cmdName, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
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
	home := os.Getenv("HOME")
	if home == "" {
		return fmt.Errorf("HOME is not set")
	}
	return os.Chdir(home)
}
