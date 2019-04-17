package main

import (
	"errors"
	"os"

	"github.com/sirupsen/logrus"
)

func main() {
	exitCode := 0

	err := run()
	if err != nil {
		exitCode = 1
		logrus.Errorln(err)
	}
	os.Exit(exitCode)
}

func run() error {
	return errors.New("unimplemented")
}
