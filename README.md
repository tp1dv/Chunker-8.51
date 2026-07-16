# Chunker-style downloader in Go

This repository contains a lightweight Go-based downloader scaffold inspired by the chunker project layout. It includes a downloader package, chunk/manifest structures, helper utilities, and a simple CLI entrypoint.

## Features

- Download chunks from a manifest-based source
- Retry failed HTTP requests
- Validate manifest structure before downloading
- Verify downloaded chunk hashes
- Write files atomically to avoid corruption
- Support threaded downloads

## Project structure

- src/chunk - chunk and manifest data structures
- src/helpers - shared helper functions
- src/downloader - downloader implementation
- src/compile - compile/build placeholder logic
- src/cmd - CLI entrypoint and argument parsing

## Requirements

- Go 1.21 or newer

## Installation

```bash
go mod tidy
```

## Build

```bash
go build -o chunker.exe ./src/cmd
```

## Usage

### Download a chunker-based manifest

```bash
./chunker.exe download --url https://cdn.fortnitearchive.com/8.51.rar --manifest 8.51.rar --output ./downloads
```

### Compile a build placeholder

```bash
./chunker.exe compile --source ./src --output ./build
```

## Notes

This project is intended as a starter scaffold and is not a complete production-ready chunk distribution system. It is designed to be extended for real CDN or manifest-driven deployments.

## License

This project is provided for learning and experimentation purposes.

# Made by Centric

coded the whole chunker 

