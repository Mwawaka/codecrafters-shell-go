package main

import (
	"os"
)

func writeToFile(filename string, data []byte, appendMode bool) error {
	flags := os.O_WRONLY | os.O_CREATE

	if appendMode {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}

	file, err := os.OpenFile(filename, flags, 0644)

	if err != nil {
		return err
	}

	defer file.Close()
	_, err = file.Write(data)
	return err
}
