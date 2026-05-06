package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	sessionstreamdoc "github.com/go-go-golems/sessionstream/pkg/doc"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := newRootCommand()
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("execute: %v", err)
	}
}

func newRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "sessionstream-systemlab",
		Short: "Run Sessionstream labs and browse Sessionstream help docs",
		Long: "sessionstream-systemlab exposes the browser-based Sessionstream lab application. " +
			"Use the serve verb to start the HTTP server. The root command also exposes Sessionstream's embedded Glazed help entries.",
	}

	helpSystem := help.NewHelpSystem()
	if err := sessionstreamdoc.AddDocToHelpSystem(helpSystem); err != nil {
		log.Fatalf("load sessionstream help docs: %v", err)
	}
	help_cmd.SetupCobraRootCommand(helpSystem, rootCmd)

	serveCommand, err := NewServeCommand()
	if err != nil {
		log.Fatalf("build serve command: %v", err)
	}
	serveCobraCommand, err := cli.BuildCobraCommandFromCommand(serveCommand,
		cli.WithParserConfig(cli.CobraParserConfig{
			ShortHelpSections: []string{schema.DefaultSlug},
		}),
	)
	if err != nil {
		log.Fatalf("build serve cobra command: %v", err)
	}
	rootCmd.AddCommand(serveCobraCommand)

	return rootCmd
}

type ServeCommand struct {
	*cmds.CommandDescription
}

type ServeSettings struct {
	Addr string `glazed:"addr"`
}

func NewServeCommand() (*ServeCommand, error) {
	commandSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}

	cmdDesc := cmds.NewCommandDescription(
		"serve",
		cmds.WithShort("Serve the Sessionstream Systemlab HTTP application"),
		cmds.WithLong(`Serve the browser-based Sessionstream Systemlab application.

The server exposes:

  /        Browser lab UI
  /docs/   Embedded Sessionstream help markdown
  /api/    Lab API endpoints

Examples:

  sessionstream-systemlab serve
  sessionstream-systemlab serve --addr :8092
`),
		cmds.WithFlags(
			fields.New(
				"addr",
				fields.TypeString,
				fields.WithDefault(":8091"),
				fields.WithHelp("Listen address for the HTTP server"),
			),
		),
		cmds.WithSections(commandSettingsSection),
	)

	return &ServeCommand{CommandDescription: cmdDesc}, nil
}

func (c *ServeCommand) Run(ctx context.Context, parsedValues *values.Values) error {
	settings := &ServeSettings{}
	if err := parsedValues.DecodeSectionInto(schema.DefaultSlug, settings); err != nil {
		return err
	}
	return runServer(ctx, settings.Addr)
}

func runServer(_ context.Context, addr string) error {
	app, err := newSystemlabServer()
	if err != nil {
		return err
	}
	log.Printf("sessionstream-systemlab listening on %s", addr)
	server := &http.Server{
		Addr:              addr,
		Handler:           app.routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	return server.ListenAndServe()
}
