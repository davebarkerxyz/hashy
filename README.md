# Hashy: fast recursive file hashing

Hashy generates md5 hashes of directory trees using multiple threads for pretty decent performance.

## 📋 Features

- Lightweight (a couple of hundred lines of Go code)
- Single binary
- Multi-threaded for better performance
- Exclude portions of the directory tree

## License

MIT

## Building hashy

[Install the Go compiler](https://go.dev/doc/install)

Clone and compile hashy:

```
git clone https://github.com/davebarkerxyz/hashy
cd hashy
go build
```

## 🧐 Why did I build this?

I was looking for an easy way to verify a backup of my (huge) MacOS home directory to make sure that the backups remained consistent when moving between media (replacing failing backups HDDs with SSDs). `find` piped to `md5` (or `md5sum` on Linux) was *ok*, but excluding paths without descending into or statting them was tricky and the syntax was opaque, to say the least.

It was also an excuse to sharpen my Go skills. There's no promise the Go code here is *pretty*, but it is simple. Mostly, hashy is just for me, but someone else might find it useful.

## ⌨️ Usage

To hash your home directory and exclude the *Library* and *.lima* subdirectories:

```
hashy ~/ -exclude ~/Library,~/.lima
```

To run with only a single worker:

```
hashy ~/ -workers 1
```

You can see the full usage information with `hashy -h`

## 📖 Output example

```
dave@purah hashy % ./hashy ~/Downloads/Archive
a3c0bd543d9c4224a1af07527a88e497 /Users/dave/Downloads/Archive/JetBrainsMono-2.242/OFL.txt
08fd688879edce6105e388211d976c3c /Users/dave/Downloads/Archive/JetBrainsMono-2.242/.DS_Store
e22968d5f6c9cfde83dbb1e2e984dc82 /Users/dave/Downloads/Archive/JetBrainsMono-2.242/fonts/.DS_Store
177ff1d368916a567c896c0c3a2d7bc4 /Users/dave/Downloads/Archive/JetBrainsMono-2.242/AUTHORS.txt
...
```

## 🐆 Performance

On my M1 MacBook Pro 14 (2021) internal SSD, hashing my 278GB home directory under the default configuration (workers = GOMAXPROCS, in my case that was 10), it took 6m57s.

Setting workers to 1 (single-threading the hashing, but leaving the main thread to handle setup and cleanup), hashing the same tree took 16m02s.

*Note: testing wasn't under any sort of controlled conditions (the MacBook is my daily driver and had been running for a few hours with several applications running during testing, and the files in my home directory vary from a few bytes to tens of gigabytes), but it's probably representative of performance under typical use.*

Performance could have been further improved if I could avoid `stat`ing every file before opening, but this was unavoidable as opening a named pipe causes `os.Open()` to block and never return, blocking a goroutine until you kill the process. The only way we can avoid that is by `stat`ing each file and making sure we only open regular files.

## 🖊️ Notes

Symlinks are ignored and not followed. Non-regular files (named pipes, etc) are also ignored.
