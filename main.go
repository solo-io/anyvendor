package main

import (
	"log"

	"github.com/solo-io/protodep/pkg/cli"
)

func main() {
	app := cli.RootCmd()
	if err := app.Execute(); err != nil {
		log.Fatalf("error executing protodep cli: %+v", err)
	}
}
