package doc

import (
	"testing"

	"github.com/go-go-golems/glazed/pkg/help"
	"github.com/stretchr/testify/require"
)

func TestAddDocToHelpSystem(t *testing.T) {
	helpSystem := help.NewHelpSystem()
	require.NoError(t, AddDocToHelpSystem(helpSystem))
}
