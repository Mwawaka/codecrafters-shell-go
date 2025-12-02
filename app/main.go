package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("$ ")
		command, err := reader.ReadString('\n')

		if err != nil {
			fmt.Fprintln(os.Stderr, "error reading input:", err)
			os.Exit(1)
		}

		if command[:len(command)-1] == "exit"{
			os.Exit(0)
		}

		fmt.Printf("%s: command not found\n", command[:len(command)-1])
	}
}
