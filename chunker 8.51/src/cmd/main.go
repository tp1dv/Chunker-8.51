package main

import (
	"acid/chunker/src/compile"
	"acid/chunker/src/downloader"
	"fmt"
	"os"
)

func main() {
	opts, err := parseOptions(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	switch opts.Command {
	case "download":
		d := downloader.NewDownloader(opts.URL, opts.OutputDir, opts.CompileDir)
		manifest, err := d.FetchManifest(opts.Manifest, opts.OutputDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		d.Manifest = manifest
		if err := d.DownloadThreaded(opts.Threads); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "compile":
		compiler := compile.NewCompiler()
		if err := compiler.Build(opts.SourceDir, opts.OutputDir); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	default:
		fmt.Println("usage: chunker <download|compile>")
	}
}
