package main

import (
	"log"
	"os"

	"github.com/canpok1/vox-radio/internal/cli"
	"github.com/spf13/cobra/doc"
)

func main() {
	outDir := "docs/cli"
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatalf("mkdir %s: %v", outDir, err)
	}

	root := cli.NewRootCmd()
	root.DisableAutoGenTag = true

	if err := doc.GenMarkdownTree(root, outDir); err != nil {
		log.Fatalf("gen docs: %v", err)
	}
}
