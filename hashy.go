package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

func main() {
	path := "./"
	if len(os.Args) > 1 {
		if os.Args[1] == "-h" {
			printUsage()
			return
		}
		path = os.Args[1]
	}

	hashDir(path)
}

func printUsage() {
	fmt.Printf(`Hash every file in supplied path, writing the hash to stdout.

Usage: %s [-h] path

-h      Display this help message
path    Path to walk (default: ./)

`, os.Args[0])
}

func hashDir(dirPath string) {
	filepath.WalkDir(dirPath, func(path string, dir fs.DirEntry, err error) error {
		if dir.IsDir() {
			return nil
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error walking %s at %s: %s", dirPath, path, err)
			return nil
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

		fmt.Fprintf(os.Stderr, "Error reading %s: %s", path, err)
	}

	fmt.Printf("%x %s\n", hasher.Sum(nil), path)
}
