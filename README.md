# Hashy: fast recursive file hashing

Hashy generates md5 hashes of directory trees using multiple threads for pretty decent performance.

## 📋 Features

- Lightweight (a couple of hundred lines of Go code)
- Single binary
- Multi-threaded for better performance
- Exclude portions of the directory tree

## License

MIT

## 🧐 Why did I build this?

I was looking for an easy way to verify a backup of my (huge) MacOS home directory to make sure that the backups remained consistent when moving between media (replacing failing backups HDDs with SSDs). `find` piped to `md5` (or `md5sum` on Linux) was *ok*, but excluding paths without descending into or statting them was tricky and the syntax was opaque, to say the least.

It was also an excuse to sharpen my Go skills (though there's no promise the Go code here is any *good*, but the tool fulfils my requirements). Mostly hashy is just for me, but someone else might find it useful.

## ⌨️ Usage

To hash your home directory and exclude the *Library* and *.lima* subdirectories:

```
hashy ~/ -exclude ~/Library,~/.lima
```

To run with only a single worker:

```
hashy ~/ -workers 1`
```

You can see the full usage information with `hashy -h`

## 🐆 Performance

On my M1 MacBook Pro 14 (2021) internal SSD, hashing my 278GB home directory under the default configuration (workers = GOMAXPROCS, in my case that was 10), it took 6m57s.

Setting workers to 1 (single-threading the hashing, but leaving the main thread to handle setup and cleanup), hashing the same tree took 16m02s.

Performance could possibly have been improved if I could avoid `stat`ing every file before opening, but this was unavoidable as opening a named pipe causes `os.Open()` to block and never return, blocking a goroutine until you kill the process. The only way we can avoid that is by `stat`ing each file and making sure we only open regular files.

## 🖊️ Notes

Symlinks are ignored and not followed. Non-regular files (named pipes, etc) are also ignored.
