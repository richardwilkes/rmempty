package main

import (
	"cmp"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/richardwilkes/toolbox/atexit"
	"github.com/richardwilkes/toolbox/cmdline"
	"github.com/richardwilkes/toolbox/collection/dict"
	"github.com/yookoala/realpath"
)

func main() {
	cmdline.AppName = "Remove Empty Directories"
	cmdline.AppVersion = "1.0.0"
	cmdline.CopyrightHolder = "Richard Wilkes"
	cmdline.CopyrightStartYear = "2018"
	cmdline.License = "Mozilla Public License Version 2.0"
	cl := cmdline.New(true)
	cl.UsageSuffix = "dirs..."
	var remove bool
	cl.NewGeneralOption(&remove).SetName("delete").SetSingle('d').SetUsage("Delete all empty directories found")
	paths := cl.Parse(os.Args[1:])

	// If no paths specified, use the current directory
	if len(paths) == 0 {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Println("unable to determine current working directory.")
			atexit.Exit(1)
		}
		paths = append(paths, wd)
	}

	// Determine the actual root paths and prune out paths that are a subset of another
	set := make(map[string]struct{})
	for _, path := range paths {
		actual, err := realpath.Realpath(path)
		if err != nil {
			fmt.Printf("unable to determine real path for '%s'.\n", path)
			atexit.Exit(1)
		}
		if _, exists := set[actual]; !exists {
			add := true
			for one := range set {
				prefixed := strings.HasPrefix(rel(one, actual), "..")
				if prefixed != strings.HasPrefix(rel(actual, one), "..") {
					if prefixed {
						delete(set, one)
					} else {
						add = false
						break
					}
				}
			}
			if add {
				set[actual] = struct{}{}
			}
		}
	}

	// Find all directories
	var dirs []string
	for root := range maps.Keys(set) {
		if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.Type().IsDir() {
				if d.Name() == ".git" {
					return fs.SkipDir
				}
				dirs = append(dirs, path)
			}
			return nil
		}); err != nil {
			fmt.Println(err)
			atexit.Exit(1)
		}
	}

	// Sort directories by length and then by name
	slices.SortFunc(dirs, func(a, b string) int {
		result := cmp.Compare(len(b), len(a))
		if result == 0 {
			return cmp.Compare(a, b)
		}
		return result
	})

	// Collect empty directories
	var empties []string
	var dsStores []bool
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			fmt.Printf("unable to read directory '%s'.\n", dir)
			atexit.Exit(1)
		}
		m := make(map[string]fs.DirEntry)
		for _, entry := range entries {
			m[filepath.Join(dir, entry.Name())] = entry
		}
		for _, empty := range empties {
			delete(m, empty)
		}
		if len(m) == 0 {
			empties = append(empties, dir)
			dsStores = append(dsStores, false)
			continue
		}
		if len(m) != 1 {
			continue
		}
		entry := dict.Values(m)[0]
		if entry.Type().IsRegular() && entry.Name() == ".DS_Store" {
			empties = append(empties, dir)
			dsStores = append(dsStores, true)
		}
	}

	extra := ""
	if remove {
		extra = " (will be removed)"
	}
	fmt.Printf("Empty directories%s:\n", extra)
	for i, one := range empties {
		fmt.Println(one)
		if remove {
			if dsStores[i] {
				if err := os.Remove(filepath.Join(one, ".DS_Store")); err != nil {
					fmt.Printf("unable to remove file '%s/.DS_Store'.\n", one)
					atexit.Exit(1)
				}
			}
			if err := os.Remove(one); err != nil {
				fmt.Printf("unable to remove directory '%s'.\n", one)
				atexit.Exit(1)
			}
		}
	}

	atexit.Exit(0)
}

func rel(base, target string) string {
	path, err := filepath.Rel(base, target)
	if err != nil {
		fmt.Println(err)
		atexit.Exit(1)
	}
	return path
}
