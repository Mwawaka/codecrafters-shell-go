package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/user"
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
		cmdName := parts[0]

		if cmdName == "exit" {
			exit()
		}

		if cmdName == "cd" {
			if err := cd(strings.Fields(command[:len(command)-1])[1:]); err != nil {
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

		if err := externalPrograms(cmdName, parts[1:]); err != nil {
			fmt.Printf("%s: command not found\n", cmdName)
		}
	}
}

func tokenizer(command string) []string {
	fmt.Println("tokenizer: ", command)
	currentToken := ""
	tokens := []string{}
	inQuote := false

	for _, r := range command {
		if r == '\'' {
			inQuote = !inQuote
			continue
		} else if r == ' ' && !inQuote {
			tokens = append(tokens, currentToken)
			currentToken = ""
			continue
		} else {
			currentToken += string(r)
		}

	}

	tokens = append(tokens, currentToken)
	fmt.Println("Token Length: ", len(tokens))
	return tokens
}

func exit() {
	os.Exit(0)
}

func echo(args []string) (string, error) {
	// fmt.Println("args: ", args)
	// fmt.Println("len: ", len(args))
	newArgs := []string{}
	for _, arg := range args {
		// fmt.Println("arg: ", arg)
		// fmt.Println("len: ", len(arg))
		if strings.Contains(arg, `'`) {
			quoteInx := strings.Index(arg, `'`)
			newArgs = append(newArgs, arg[quoteInx+1:len(arg)-1])
			continue
		}
		newArgs = append(newArgs, arg)
	}
	return strings.Join(newArgs, " "), nil
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

func externalPrograms(file string, args []string) error {
	// TODO: support for interactive programs
	if _, err := exec.LookPath(file); err != nil {
		return err
	}

	cmd := exec.Command(file, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
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
		return chDirToHomeV2()
	}

	if err := os.Chdir(args[0]); err != nil {
		return fmt.Errorf("cd: %s: No such file or directory", args[0])

	}
	return nil
}

func ChDirToHome() error {
	// TODO: recommended, portable
	u, err := user.Current()
	if err != nil {
		return err
	}
	return os.Chdir(u.HomeDir)
}

func chDirToHomeV2() error {
	// TODO: ensure turbo tests pass
	return os.Chdir(os.Getenv("HOME"))
}

// TODO: unique error for 2 args
// if len(args) == 2 {
// 	return fmt.Errorf("cd: %s: string not in pwd", args[1])
// }
