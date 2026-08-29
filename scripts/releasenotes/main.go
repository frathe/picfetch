// Command releasenotes builds GitHub release notes from todos.md's ## Done
// section. Run from the repository root:
//
//	go run ./scripts/releasenotes --prev 0.2.7 --next 0.2.8
//	go run ./scripts/releasenotes --prev 0.2.7 --next 0.2.8 --write .github/release-notes.md --clear-done
package main

import (
	"flag"
	"fmt"
	"os"
)

const todosPath = "todos.md"

func main() {
	if err := run(os.Args[1:]); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("releasenotes", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	prev := fs.String("prev", "", "previous version (with or without v prefix)")
	next := fs.String("next", "", "new version (with or without v prefix)")
	write := fs.String("write", "", "write notes to this path instead of stdout")
	clearDone := fs.Bool("clear-done", false, "replace todos.md ## Done items with the empty category skeleton")
	if err := fs.Parse(args); err != nil {
		return err
	}
	extract := *prev != "" || *next != "" || *write != ""
	if extract && (*prev == "" || *next == "") {
		return fmt.Errorf("usage: releasenotes --prev VERSION --next VERSION [--write PATH] [--clear-done]")
	}
	if !extract && !*clearDone {
		return fmt.Errorf("usage: releasenotes --prev VERSION --next VERSION [--write PATH] [--clear-done]")
	}
	if _, err := os.Stat(todosPath); err != nil {
		return fmt.Errorf("run from the repository root")
	}

	raw, err := os.ReadFile(todosPath)
	if err != nil {
		return err
	}
	todos := string(raw)

	if extract {
		notes, err := Build(todos, *prev, *next)
		if err != nil {
			return err
		}
		if *write != "" {
			if err := os.WriteFile(*write, []byte(notes), 0o644); err != nil {
				return err
			}
		} else {
			fmt.Print(notes)
		}
	}
	if *clearDone {
		out, err := ClearDone(todos)
		if err != nil {
			return err
		}
		if err := os.WriteFile(todosPath, []byte(out), 0o644); err != nil {
			return err
		}
	}
	return nil
}
