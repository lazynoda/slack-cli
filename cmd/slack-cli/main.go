package main

import (
	"fmt"
	"io"
	"os"

	"github.com/alecthomas/kong"

	"slack-cli/internal/app"
	"slack-cli/internal/cli"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, out io.Writer, errOut io.Writer) int {
	command := cli.New()
	parser, err := kong.New(
		command,
		kong.Name("slack-cli"),
		kong.Description("Slack Web API CLI. Version ${version}."),
		kong.UsageOnError(),
		kong.Writers(out, errOut),
		kong.Vars(map[string]string{
			"version": version,
			"commit":  commit,
			"date":    date,
		}),
	)
	if err != nil {
		_, _ = fmt.Fprintln(errOut, err)
		return 2
	}

	kctx, err := parser.Parse(args)
	if err != nil {
		_, _ = fmt.Fprintln(errOut, err)
		return 2
	}

	settings, err := command.Globals.Settings()
	if err != nil {
		_, _ = fmt.Fprintln(errOut, err)
		return 2
	}

	ctx, err := app.NewContext(settings, out, errOut)
	if err != nil {
		_, _ = fmt.Fprintln(errOut, err)
		return 1
	}

	if err := kctx.Run(ctx); err != nil {
		_, _ = fmt.Fprintln(errOut, err)
		return 1
	}

	return 0
}
