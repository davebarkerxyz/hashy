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
	"runtime"
	"sync"
)

var defaultWorkerCount = runtime.GOMAXPROCS(0)

func main() {
	dirPath := "./"
	var workerCount int

	flag.IntVar(&workerCount, "workers", defaultWorkerCount, "number of workers")
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

	hashDir(dirPath, workerCount)
}

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

func printUsage() {
	fmt.Printf(`Hash every file in supplied path, writing the hash to stdout.

Usage: %s [-h] <path>

-h          Display this help message
-workers    Number of workers (default: %d)
path        Path to walk (default: ./)

`, os.Args[0], defaultWorkerCount)
	os.Exit(0)
}

func hashDir(dirPath string, workerCount int) {
	jobs := make(chan string, workerCount)
	wg := new(sync.WaitGroup)

	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go hashWorker(jobs, wg)
	}

	filepath.WalkDir(dirPath, func(path string, dir fs.DirEntry, err error) error {
		if dir.IsDir() {
			return nil
		}

		if err != nil {
			die("Error walking %s at %s: %s", dirPath, path, err)
		}

		jobs <- path

		return nil
	})
	close(jobs)

	wg.Wait()
}

func hashWorker(jobs chan string, wg *sync.WaitGroup) {
	for path := range jobs {
		fmt.Print(hashFile(path))
	}
	wg.Done()
}

func hashFile(path string) string {
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening %s: %s\n", path, err)
		return ""
	}
	defer f.Close()

	hasher := md5.New()
	if _, err := io.Copy(hasher, f); err != nil {
		// Check if the file is a directory
		// This can happen if the file being walked is a symlink that resolves to a directory
		stat, _ := f.Stat()
		if stat.IsDir() {
			return ""
		}

		die("Error reading %s: %s\n", path, err)
	}

	return fmt.Sprintf("%x %s\n", hasher.Sum(nil), path)
}
