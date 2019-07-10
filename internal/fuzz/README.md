# Fuzzing

Fuzzing support using [go-fuzz](https://github.com/dvyukov/go-fuzz).

Basic operation:

```sh
$ go install github.com/dvyukov/go-fuzz/go-fuzz github.com/dvyukov/go-fuzz/go-fuzz-build
$ cd internal/fuzz/{fuzzer}
$ GOFUZZ111MODULE=on go-fuzz-build .
$ go-fuzz
```
