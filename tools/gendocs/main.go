package main

import (
	"log"

	"github.com/canpok1/vox-radio/internal/cli"
	"github.com/canpok1/vox-radio/internal/fileio"
	"github.com/spf13/cobra/doc"
)

func main() {
	outDir := "docs/cli"
	if err := fileio.EnsureDir(outDir); err != nil {
		log.Fatalf("mkdir %s: %v", outDir, err)
	}

	root := cli.NewRootCmd()
	if err := doc.GenMarkdownTree(root, outDir); err != nil {
		log.Fatalf("gen docs: %v", err)
	}
}
