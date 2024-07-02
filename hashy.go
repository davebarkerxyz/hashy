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

func main() {
	dirPath := "./"
	var workerCount int
	var exclude string

	flag.IntVar(&workerCount, "workers", defaultWorkerCount, "number of workers")
	flag.StringVar(&exclude, "exclude", "", "list of directories to exclude")
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

-h          Display this help message
-workers    Number of workers (default: %d)
-exclude    Comma separated list of directories to exclude
path        Path to walk (default: ./)

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
