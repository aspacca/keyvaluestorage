package main

import (
	"fmt"
	"github.com/aspacca/keyvaluestorage/http"
	"github.com/aspacca/keyvaluestorage/storage"
	"github.com/minio/cli"
)

var version = "0.1"
var helpTemplate = `NAME:
{{.Name}} - {{.Usage}}

DESCRIPTION:
{{.Description}}

USAGE:
{{.Name}} {{if .Flags}}[flags] {{end}}command{{if .Flags}}{{end}} [arguments...]

COMMANDS:
{{range .Commands}}{{join .Names ", "}}{{ "\t" }}{{.Usage}}
{{end}}{{if .Flags}}
FLAGS:
{{range .Flags}}{{.}}
{{end}}{{end}}
VERSION:
` + version +
	`{{ "\n"}}`

var globalFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "listener",
		Usage: "0.0.0.0:8080",
		Value: "0.0.0.0:8080",
	},
	cli.StringFlag{
		Name:  "basedir",
		Usage: "path to storage",
		Value: "",
	},
	cli.StringFlag{
		Name:  "provider",
		Usage: "fs|memory",
		Value: "",
	},
}

type cmd struct {
	*cli.App
}

func versionAction(c *cli.Context) {
	fmt.Println("Key value storage server ver")
}

func newServer() *cmd {
	app := cli.NewApp()
	app.Name = "Key value storage server"
	app.Version = version
	app.Author = "Andrea Spacca"
	app.Description = "Key value storage server"
	app.Flags = globalFlags
	app.CustomAppHelpTemplate = helpTemplate
	app.Commands = []cli.Command{
		{
			Name:   "version",
			Action: versionAction,
		},
	}

	app.Before = func(c *cli.Context) error {
		return nil
	}

	app.Action = func(c *cli.Context) {
		options := []http.OptionFn{}
		if v := c.String("listener"); v != "" {
			options = append(options, http.Listener(v))
		}

		switch provider := c.String("provider"); provider {
		case "fs":
			if v := c.String("basedir"); v == "" {
				panic("basedir not set.")
			} else if storage, err := storage.NewFileSystemStorage(v); err != nil {
				panic(err)
			} else {
				options = append(options, http.UseStorage(storage))
			}
		case "memory":
			if v := c.String("basedir"); v == "" {
				panic("basedir not set.")
			} else if storage, err := storage.NewMemoryStorage(v); err != nil {
				panic(err)
			} else {
				options = append(options, http.UseStorage(storage))
			}
		default:
			panic("Provider not set or invalid.")
		}

		s, err := http.New(
			options...,
		)

		if err != nil {
			panic(fmt.Sprintf("Error starting server: %s\n", err))
		}

		s.Run()
	}

	return &cmd{
		App: app,
	}
}

func main() {
	app := newServer()
	app.RunAndExitOnError()
}
