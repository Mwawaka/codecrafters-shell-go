package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type CommandHandler func(args []string) (string, error)

var commands = map[string]CommandHandler{
	"echo": echo,
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
	return strings.Join(args, " "),nil
}
