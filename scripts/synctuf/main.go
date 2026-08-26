// Command synctuf checks or refreshes the GitHub TUF root embed used by
// the in-app updater. Run from the repository root:
//
//	go run ./scripts/synctuf --check
//	go run ./scripts/synctuf --write
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/frathe/picfetch/internal/update"
)

const embedPath = "internal/update/embed/tuf-repo.github.com/root.json"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: synctuf --check|--write")
	}
	if _, err := os.Stat(embedPath); err != nil {
		return fmt.Errorf("run from the repository root")
	}
	switch args[0] {
	case "--check":
		root, err := os.ReadFile(embedPath)
		if err != nil {
			return err
		}
		return update.CheckEmbeddedRoot(root, time.Now())
	case "--write":
		hc := &http.Client{Timeout: 30 * time.Second}
		_, err := update.SyncGitHubRoot(context.Background(), embedPath, hc)
		return err
	default:
		return fmt.Errorf("usage: synctuf --check|--write")
	}
}
