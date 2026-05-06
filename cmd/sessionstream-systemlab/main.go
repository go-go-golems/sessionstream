package main

import (
	"log"
	"net/http"
	"time"

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
	var addr string

	rootCmd := &cobra.Command{
		Use:   "sessionstream-systemlab",
		Short: "Run the Sessionstream browser lab and serve Sessionstream help docs",
		Long: "sessionstream-systemlab runs the browser-based Sessionstream lab application. " +
			"It also exposes Sessionstream's embedded Glazed help entries through the CLI help system and over HTTP under /docs/.",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runServer(addr)
		},
	}

	rootCmd.Flags().StringVar(&addr, "addr", ":8091", "listen address")

	helpSystem := help.NewHelpSystem()
	if err := sessionstreamdoc.AddDocToHelpSystem(helpSystem); err != nil {
		log.Fatalf("load sessionstream help docs: %v", err)
	}
	help_cmd.SetupCobraRootCommand(helpSystem, rootCmd)

	return rootCmd
}

func runServer(addr string) error {
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
