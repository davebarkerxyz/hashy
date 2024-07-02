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
	"strings"
	"sync"
)

var defaultWorkerCount = runtime.GOMAXPROCS(0)
var showAllErrors bool

type unsupportedFileError struct {
	path   string
	reason string
}

func (e unsupportedFileError) Error() string {
	return fmt.Sprintf("unhashable file %s: %s", e.path, e.reason)
}

func main() {
	dirPath := "./"
	var workerCount int
	var exclude string

	flag.IntVar(&workerCount, "workers", defaultWorkerCount, "number of workers")
	flag.StringVar(&exclude, "exclude", "", "list of directories to exclude")
	flag.BoolVar(&showAllErrors, "show-errors", false, "show all errors (including unhashable file types)")
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

	var excludeRaw = strings.Split(exclude, ",")
	var excludeList []string

	if exclude != "" {
		for _, dir := range excludeRaw {
			apath, _ := filepath.Abs(dir + "/")
			excludeList = append(excludeList, apath)
		}
	}

	hashDir(dirPath, workerCount, excludeList)
}

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

func printUsage() {
	fmt.Printf(`Hash every file in supplied path, writing the hash to stdout.

Usage: hashy [-h] <path> [-workers 4] [-exclude path1,path2]

-h             Display this help message
-workers       Number of workers (default: %d)
-exclude       Comma separated list of directories to exclude
-show-errors   Show all errors (including unhashable file types) (default: false)
path           Path to walk (default: ./)

For example: hashy ~/ -workers 4 -exclude ~/Library,~/.lima

`, defaultWorkerCount)
	os.Exit(0)
}

func hashDir(dirPath string, workerCount int, excludeList []string) {
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

		for _, exclude := range excludeList {
			if strings.HasPrefix(path, exclude) {
				return nil
			}
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
		hash, err := hashFile(path)

		var badFileErr *unsupportedFileError
		if err == nil {
			fmt.Print(hash)
		} else if !errors.As(err, &badFileErr) || showAllErrors {
			// Error isn't due to trying to hash non-regular file? Print it.
			// Otherwise, ignore it (we know we can't hash sockets, etc)
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		}
	}
	wg.Done()
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		// Check if the file is a regular file - if not, ignore the error
		// as we don't support hashing sockets or other special files
		// For performance reasons, only stat files if we have an issue opening them
		finfo, statErr := os.Lstat(path)
		if statErr != nil || finfo.Mode().IsRegular() {
			return "", fmt.Errorf("error opening %s: %s", path, err)
		}
		return "", &unsupportedFileError{path, "not a regular file"}
	}
	defer f.Close()

	hasher := md5.New()
	if _, err := io.Copy(hasher, f); err != nil {
		// Check if the file is a directory
		// This can happen if the file being walked is a symlink that resolves to a directory
		stat, _ := f.Stat()
		if stat.IsDir() {
			return "", &unsupportedFileError{path, "directories cannot be hashed (possible symlink to dir)"}
		}

		die("Error reading %s: %s\n", path, err)
	}

	return fmt.Sprintf("%x %s\n", hasher.Sum(nil), path), nil
}
