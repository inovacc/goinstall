# Golang Installer

[ ] golang install cli and autoupdate:
    * check pid of installed app for update if needed

this app is a `goinstall` app that is a wrapper around `go install` plus `sqlite` database to handle all modules installed and eventually monitoring then for updates

## command to run install
```shell
goinstall https://github.com/inovacc/base-utils
goinstall git://github.com/inovacc/base-utils
goinstall ssh://github.com/inovacc/base-utils
goinstall github.com/inovacc/base-utils
```

## project structure

goinstall/
├── cmd
│   └── root.go
├── go.mod
├── go.sum
├── internal
│   ├── db
│   │   └── db.go
│   ├── installer
│   │   └── installer.go
│   └── monitor
│       └── monitor.go
├── LICENSE
├── main.go
└── README.md

## first try implementing

````go
var rootCmd = &cobra.Command{
    Use:   "goinstall <module-path>",
    Short: "Install and manage Go binaries with tracking",
    RunE: func(cmd *cobra.Command, args []string) error {
        return installer.InstallModule(args[0])
    },
}

// install/installer.go
func InstallModule(module string) error {
    cmd := exec.Command("go", "install", fmt.Sprintf("%s@latest", module))
    cmd.Env = append(os.Environ(), "GOBIN="+yourBinDir)
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("install failed: %w", err)
    }
    // Save to SQLite DB here
    return nil
}

````

## database schema

```sql
CREATE TABLE IF NOT EXISTS modules (
    id INTEGER PRIMARY KEY,
    path TEXT UNIQUE,
    version TEXT,
    installed_at DATETIME,
    pid INTEGER
);
```

## first prototype

```go
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

var dbPath = filepath.Join(os.Getenv("HOME"), ".goinstall", "modules.db")

func main() {
	rootCmd := &cobra.Command{
		Use:   "goinstall <module>",
		Short: "Install and track Go binaries",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return installModule(args[0])
		},
	}

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func installModule(module string) error {
	// Ensure DB folder exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return err
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS modules (
			id INTEGER PRIMARY KEY,
			path TEXT UNIQUE,
			installed_at DATETIME
		)
	`)
	if err != nil {
		return err
	}

	fmt.Println("Installing", module)
	cmd := exec.Command("go", "install", module+"@latest")
	cmd.Env = append(os.Environ(), fmt.Sprintf("GOBIN=%s",filepath.Join(os.Getenv("HOME"), "go"), "bin"))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	_, err = db.Exec(`INSERT OR REPLACE INTO modules (path, installed_at) VALUES (?, ?)`, module, time.Now())
	if err != nil {
		return err
	}

	fmt.Println("✅ Installed and recorded", module)
	return nil
}
```