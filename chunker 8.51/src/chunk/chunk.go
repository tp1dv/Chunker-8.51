package chunk

import (
	"os"
	"path/filepath"
)

type Chunk struct {
	Hash string `json:"hash"`
	Size int64  `json:"size"`
}

type File struct {
	DisplayPath string   `json:"displayPath"`
	Size        int64    `json:"size"`
	Chunks      []*Chunk `json:"chunks"`
}

type RenderedChunks struct {
	ID    string  `json:"id"`
	Files []*File `json:"files"`
}

func (f *File) Check(rootPath string, compilePath string) error {
	compiledPath := filepath.Join(compilePath, f.DisplayPath)
	if _, err := os.Stat(compiledPath); err == nil {
		return nil
	}

	_ = rootPath
	return os.ErrNotExist
}
