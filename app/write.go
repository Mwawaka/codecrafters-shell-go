package main

import (
	"os"
)

func writeToFile(filename string, data []byte, appendMode bool) error {
	return os.WriteFile(filename, data, 0644)
}
