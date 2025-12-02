package main

import (
	"bufio"
	"fmt"
	"os"
	"slices"
	"strings"
)

type CommandHandler func(args []string) (string, error)

var commands = map[string]CommandHandler{
	"echo": echo,
	"type": typo,
}

func main() {

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("$ ")
		command, err := reader.ReadString('\n')

		if err != nil {
			fmt.Fprintln(os.Stderr, "error reading input:", err)
			os.Exit(1)
		}

		parts := strings.Split(command[:len(command)-1], " ")
		cmdName := parts[0]

		if cmdName == "exit" {
			exit()
		}

		if handler, exists := commands[cmdName]; exists {
			out, err := handler(parts[1:])
			if err != nil {
				fmt.Fprintln(os.Stderr, "error executing the command:", err)
				continue
			}
			fmt.Println(out)
			continue
		}

		fmt.Printf("%s: command not found\n", cmdName)

	}
}

func exit() {
	os.Exit(0)
}

func echo(args []string) (string, error) {
	return strings.Join(args, " "), nil
}

func typo(args []string) (string, error) {
	types := []string{"echo", "exit", "type"}
	msg := make([]string, len(args))
	for i, arg := range args {
		if !slices.Contains(types, arg) {
			// msg[i] = fmt.Sprintf("%s: not found", arg)
			fmt.Println(os.LookupEnv("PATH"))
			continue
		}
		msg[i] = fmt.Sprintf("%s is a shell builtin", arg)
	}
	return strings.Join(msg, "\n"), nil
}

// TODO://consult with claude
