package main

import (
	"cmp"
	"flag"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/richardwilkes/toolbox/v2/xflag"
	"github.com/richardwilkes/toolbox/v2/xos"
	"github.com/yookoala/realpath"
)

func main() {
	xos.AppName = "Remove Empty Directories"
	xos.AppVersion = "1.1.0"
	xos.CopyrightHolder = "Richard Wilkes"
	xos.CopyrightStartYear = "2018"
	xos.License = "Mozilla Public License Version 2.0"
	xflag.SetUsage(nil, "", "[dir]...")
	remove := flag.Bool("delete", false, "Delete all empty directories found")
	xflag.AddVersionFlags()
	xflag.Parse()
	paths := flag.Args()

	// If no paths specified, use the current directory
	if len(paths) == 0 {
		wd, err := os.Getwd()
		xos.ExitIfErr(err)
		paths = append(paths, wd)
	}

	// Determine the actual root paths and prune out paths that are a subset of another
	set := make(map[string]struct{})
	for _, path := range paths {
		actual, err := realpath.Realpath(path)
		xos.ExitIfErr(err)
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
		xos.ExitIfErr(filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
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
		}))
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
		xos.ExitIfErr(err)
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
		entry := slices.Collect(maps.Values(m))[0]
		if entry.Type().IsRegular() && entry.Name() == ".DS_Store" {
			empties = append(empties, dir)
			dsStores = append(dsStores, true)
		}
	}

	extra := ""
	if *remove {
		extra = " (will be removed)"
	}
	fmt.Printf("Empty directories%s:\n", extra)
	for i, one := range empties {
		fmt.Println(one)
		if *remove {
			if dsStores[i] {
				xos.ExitIfErr(os.Remove(filepath.Join(one, ".DS_Store")))
			}
			xos.ExitIfErr(os.Remove(one))
		}
	}

	xos.Exit(0)
}

func rel(base, target string) string {
	path, err := filepath.Rel(base, target)
	xos.ExitIfErr(err)
	return path
}
