package main

import (
	"errors"
	"os"
)

func main() {
	exitCode := 0

	err := run()
	if err != nil {
		exitCode = 1
	}
	os.Exit(exitCode)
}

func run() error {
	return errors.New("unimplemented")
}
