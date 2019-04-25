package main

import (
	"os"

	"github.com/sirupsen/logrus"

	"github.com/coreos/airlock/internal/cli"
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
	cmd, err := cli.Init()
	if err != nil {
		return err
	}

	return cmd.Execute()
}
