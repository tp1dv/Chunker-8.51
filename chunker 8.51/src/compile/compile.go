package compile

import (
	"fmt"
	"os"
	"path/filepath"
)

type Compiler struct{}

func NewCompiler() *Compiler {
	return &Compiler{}
}

func (c *Compiler) Build(sourceDir string, outputDir string) error {
	if sourceDir == "" {
		sourceDir = "."
	}
	if outputDir == "" {
		outputDir = filepath.Join(sourceDir, "build")
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}

	manifestPath := filepath.Join(outputDir, "manifest.json")
	content := fmt.Sprintf("{\"source\":\"%s\",\"build\":\"%s\"}\n", sourceDir, outputDir)
	return os.WriteFile(manifestPath, []byte(content), 0o644)
}
