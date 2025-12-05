package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func handleError(err error, msg string) {
	fmt.Fprintln(os.Stderr, msg, err)
}

func writeFile(filename, msg string) {
	byteSlice := []byte(msg)
	path := filepath.Join(filename)

	if err := os.WriteFile(path, byteSlice, 0644); err != nil {
		handleError(err, "error writing to file")
	}
}
