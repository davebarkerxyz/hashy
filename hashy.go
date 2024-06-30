package main

import (
	"crypto/md5"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

func main() {
	dirPath := "./"
	var workers int

	flag.IntVar(&workers, "workers", 1, "number of workers")
	flag.Usage = printUsage
	flag.Parse()

	args := flag.Args()
	if flag.NArg() > 1 {
		printUsage()
	} else if flag.NArg() == 1 {
		dirPath = args[0]
	}

	_, err := os.Stat(dirPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			die("Error: %s does not exist.\n", dirPath)
		} else {
			die("Error reading %s: %s\n", dirPath, err)
		}
	}

	hashDir(dirPath)
}

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

func printUsage() {
	fmt.Printf(`Hash every file in supplied path, writing the hash to stdout.

Usage: %s [-h] <path>

-h          Display this help message
-workers    Number of workers (default: 1)
path        Path to walk (default: ./)

`, os.Args[0])
	os.Exit(0)
}

func hashDir(dirPath string) {
	filepath.WalkDir(dirPath, func(path string, dir fs.DirEntry, err error) error {
		if dir.IsDir() {
			return nil
		}

		if err != nil {
			die("Error walking %s at %s: %s", dirPath, path, err)
		}

		hashFile(path)

		return nil
	})
}

func hashFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening %s: %s", path, err)
		return
	}
	defer f.Close()

	hasher := md5.New()
	if _, err := io.Copy(hasher, f); err != nil {
		// Check if the file is a directory
		// This can happen if the file being walked is a symlink that resolves to a directory
		stat, _ := f.Stat()
		if stat.IsDir() {
			return
		}

		die("Error reading %s: %s", path, err)
	}

	fmt.Printf("%x %s\n", hasher.Sum(nil), path)
}
