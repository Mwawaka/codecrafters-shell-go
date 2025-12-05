package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
)

func handleError(err error, msg string) {
	fmt.Fprintln(os.Stderr, msg, err)
}

func writeFile(filename, msg string) {
	byteSlice := []byte(msg)
	// cwd, err := pwd([]string{})

	// if err != nil {
	// 	handleError(err, err.Error())
	// }

	path := filepath.Join(filename)

	if err := os.WriteFile(path, byteSlice, 0644); err != nil {
		handleError(err, "error writing to file")
	}
}

func WriteBytes(filename, msg string) {
	byteSlice := []byte(msg)
	file, closer, err := CreateFile(filename)

	if err != nil {
		handleError(err, err.Error())
	}

	defer closer()

	if _, err = file.Write(byteSlice); err != nil {
		handleError(err, err.Error())
	}
	file.Sync() //forces buffered data to be written to the disk
}

func WriteString(filename, msg string) {

	file, closer, err := CreateFile(filename)

	if err != nil {
		handleError(err, err.Error())
	}

	defer closer()

	if _, err = file.WriteString(msg); err != nil {
		handleError(err, err.Error())
	}
}

func WriteBuffer(filename, msg string) {
	file, closer, err := CreateFile(filename)

	if err != nil {
		handleError(err, err.Error())
	}
	defer closer()

	writer := bufio.NewWriter(file)
	if _, err = writer.WriteString(msg); err != nil {
		handleError(err, err.Error())
	}
	writer.Flush() // forces data from Go memory to underlying OS buffers
}

func CreateFile(filename string) (*os.File, func(), error) {
	cwd, err := pwd([]string{})

	if err != nil {
		handleError(err, err.Error())
	}

	file, err := os.Create(filepath.Join(cwd, filename))
	if err != nil {
		return nil, nil, err
	}
	return file, func() {
		file.Close()
	}, nil
}

/*
os.WriteFile          → 100%
file.Write([]byte)    → 98–99%
file.WriteString      → 98–99%
bufio.Writer (single) → 70–85%   ← extra allocation + method calls
*/
