# go-test

## go mod and work
```bash
$ go mod init github.com/seungkyua/go-test
$ go mod tidy
$ go mod vendor

$ go work init
$ go work use .
```

## go test
```bash
$ go test interface/embed/calculate_test.go
ok      command-line-arguments  0.390s
```