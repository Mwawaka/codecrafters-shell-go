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
