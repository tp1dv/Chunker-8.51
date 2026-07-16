package main

import (
    "fmt"
    "os"
    "path/filepath"
)

func main() {
    targetDir := filepath.Join(os.Getenv("USERPROFILE"), "Desktop", "Fortnite_Build_14.30")
    if err := os.MkdirAll(targetDir, 0o755); err != nil {
        panic(err)
    }

    files := []struct {
        name    string
        content string
    }{
        {
            name: "README.txt",
            content: "Fortnite Build Installer v14.30\nCreated by Go installer scaffold\nThis package is a placeholder demonstration only.\n",
        },
        {
            name: "install-info.json",
            content: `{"name":"Fortnite Build 14.30","type":"placeholder-installer","createdBy":"Go"}`,
        },
    }

    for _, file := range files {
        path := filepath.Join(targetDir, file.name)
        if err := os.WriteFile(path, []byte(file.content), 0o644); err != nil {
            panic(err)
        }
    }

    fmt.Printf("Installer completed. Files created in %s\n", targetDir)
}