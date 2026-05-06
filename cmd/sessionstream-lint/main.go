package main

import (
	"github.com/go-go-golems/sessionstream/pkg/analysis/sessionstreamschema"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(sessionstreamschema.Analyzer)
}
