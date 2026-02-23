package redirect

import (
	"strconv"
	"strings"
)

const (
	FdStdin  int = 0
	FdStdout int = 1
	FdStderr int = 2
)

func Redirect(token string) (fd int, isAppend, isRedirect bool) {
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

func Pipe(tokens []string) [][]string {
	var commands [][]string
	var current []string

	for _, token := range tokens {

		if token == "|" {
			if len(current) > 0 {
				commands = append(commands, current)
				current = nil
			}

			continue
		}

		current = append(current, token)
	}

	if len(current) > 0 {
		commands = append(commands, current)
	}

	// 	fmt.Println("Commands: ",commands)
	return commands
}
