package main

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"flag"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/davebarkerxyz/hashy/internal/util"
)

var defaultWorkerCount = runtime.GOMAXPROCS(0)
var showAllErrors bool
var debug bool
var version = "latest"

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
	var versionFlag bool
	var algorithm string

	flag.IntVar(&workerCount, "workers", defaultWorkerCount, "")
	flag.IntVar(&workerCount, "w", defaultWorkerCount, "")
	flag.StringVar(&exclude, "exclude", "", "")
	flag.StringVar(&exclude, "x", "", "")
	flag.StringVar(&algorithm, "algorithm", "md5", "")
	flag.StringVar(&algorithm, "a", "md5", "")
	flag.BoolVar(&showAllErrors, "show-errors", false, "")
	flag.BoolVar(&debug, "debug", false, "")
	flag.BoolVar(&versionFlag, "version", false, "")
	flag.BoolVar(&versionFlag, "v", false, "")
	flag.Usage = printUsage
	flag.Parse()

	if versionFlag {
		fmt.Printf("hashy version: %s\n", version)
		os.Exit(0)
	}

	// Check hash algorith is valid
	_, hInitErr := getHasher(algorithm)
	if hInitErr != nil {
		util.Die("failed to initialise hasher: %s\n", hInitErr)
	}

	args := flag.Args()
	if flag.NArg() > 1 {
		printUsage()
	} else if flag.NArg() == 1 {
		dirPath = args[0]
	}

	_, err := os.Stat(dirPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			util.Die("Error: %s does not exist.\n", dirPath)
		} else {
			util.Die("Error reading %s: %s\n", dirPath, err)
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

	dPrint("\033[2J")

	hashDir(dirPath, workerCount, excludeList, algorithm)
}

func dPrint(format string, args ...any) bool {
	if debug {
		util.TermPrint(format, args...)
		return true
	} else {
		return false
	}
}

func getHasher(algo string) (hash.Hash, error) {
	switch algo {
	case "md5":
		return md5.New(), nil
	case "sha1":
		return sha1.New(), nil
	case "sha256":
		return sha256.New(), nil
	case "sha512":
		return sha512.New(), nil
	}

	return nil, fmt.Errorf("unsupported algorithm %s. see hashy -h for list of supported options", algo)
}

func printUsage() {
	fmt.Printf(`Recursively hash every file in supplied path, writing the hash to stdout.

Usage: hashy [-hv] [-a algorithm] [-w number] [-x path1,path2] path

-a,algorithm algorithm    hash algorithm: md5, sha1, sha256, sha512 (default: md5)
-debug                    enable debug output
-h                        show this help message
-show-errors              show normally-suppressed errors (like skipping non-regular files)
-v,-version               show version number
-w,-workers number        number of workers (default: %d)
-x,exclude path1,path1    comma-separated list of directories to exclude

Example: hashy -a sha256 -x $HOME/Library,$HOME/.lima ~/

`, defaultWorkerCount)
	os.Exit(0)
}

func hashDir(dirPath string, workerCount int, excludeList []string, algorithm string) {
	jobs := make(chan string, workerCount)
	wg := new(sync.WaitGroup)

	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go hashWorker(w, jobs, wg, algorithm)
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
			util.Die("Error walking %s at %s: %s", dirPath, path, err)
		}

		jobs <- path

		return nil
	})
	close(jobs)

	wg.Wait()
}

func hashWorker(id int, jobs chan string, wg *sync.WaitGroup, algorithm string) {
	defer wg.Done()

	for path := range jobs {
		hash, err := hashFile(path, id, algorithm)

		var badFileErr *unsupportedFileError
		if err == nil {
			if !dPrint("%d: %s", id, hash) {
				fmt.Printf("%s\n", hash)
			}
		} else if !errors.As(err, &badFileErr) || showAllErrors {
			// Error isn't due to trying to hash non-regular file? Print it.
			// Otherwise, ignore it (we know we can't hash sockets, etc)
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		}
	}

	dPrint("%d: worker done", id)
}

func hashFile(path string, id int, algorithm string) (string, error) {
	// Check if file is regular file
	// We have to do this every time despite the performance impact as named piped will
	// "open" and can be read with io.Copy but block the goroutine and never finish reading
	dPrint("%d: statting %s", id, path)
	stat, statErr := os.Lstat(path)
	if statErr != nil {
		return "", fmt.Errorf("error statting %s: %s", path, statErr)
	}

	fileMode := stat.Mode()
	dPrint("%d: %s (%t) %s", id, fileMode.String(), fileMode.IsRegular(), path)

	if !fileMode.IsRegular() {
		return "", &unsupportedFileError{path, "not a regular file"}
	}

	dPrint("%d: opening %s", id, path)
	f, err := os.Open(path)
	dPrint("%d: opened %s", id, path)
	if err != nil {
		return "", fmt.Errorf("error opening %s: %s", path, err)
	}

	defer f.Close()

	dPrint("%d: hashing %s", id, path)
	hasher, err := getHasher(algorithm)
	if err != nil {
		util.Die("failed to initialise hasher: %s\n", err)
	}

	if _, err := io.Copy(hasher, f); err != nil {
		util.Die("Error reading %s: %s\n", path, err)
	}
	dPrint("%d: hashed %s", id, path)

	return fmt.Sprintf("%x %s", hasher.Sum(nil), path), nil
}
