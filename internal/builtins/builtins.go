package builtins

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type CommandHandler func(args []string) (string, error)

// func exit() {
// 	os.Exit(0)
// }

func Echo(args []string) (string, error) {
	return strings.Join(args, " "), nil
}

func Type(builtins map[string]CommandHandler, args []string) (string, error) {

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

func Pwd(args []string) (string, error) {
	path, err := os.Getwd()

	if err != nil {
		return "", err
	}

	return path, nil
}

func Cd(args []string) error {
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


