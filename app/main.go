package main

import (
	"fmt"
	"path/filepath"

	"github.com/codecrafters-io/shell-starter-go/internal/cmd"
)

func main() {
	fmt.Println(filepath.Dir("app/berry/"))
	fmt.Println(filepath.Base("app/berry/"))
	cmd.Cmd()
}
