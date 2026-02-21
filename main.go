package main

import (
	"context"
	"log"
	"os"

	"github.com/urfave/cli/v3"
)

func main() {
	app := cli.Command{
		Name:     "codectx",
		Usage:    "AI Code Documentation Package Manager",
		Commands: []*cli.Command{},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
