package doc

import (
	"embed"
	"io/fs"

	"github.com/go-go-golems/glazed/pkg/help"
)

//go:embed *
var docFS embed.FS

// FS returns the embedded Sessionstream help documentation filesystem.
func FS() fs.FS {
	return docFS
}

// AddDocToHelpSystem loads Sessionstream help sections into a Glazed help system.
func AddDocToHelpSystem(helpSystem *help.HelpSystem) error {
	return helpSystem.LoadSectionsFromFS(docFS, ".")
}
