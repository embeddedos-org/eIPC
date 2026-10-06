# Development

## Contribution source of truth

[CONTRIBUTING](https://github.com/embeddedos-org/eIPC/blob/master/CONTRIBUTING.md)

Before proposing a change, also review the [README](https://github.com/embeddedos-org/eIPC/blob/master/README.md). Keep changes scoped, add tests appropriate to the affected behavior, and follow the repository's current automation and review requirements.

## Build and dependency inputs found

`Dockerfile`, `Makefile`, `go.mod`, `sdk/c/CMakeLists.txt`, `sdk/c/tests/CMakeLists.txt`.

## Tests found in the default-branch tree

`config/config_test.go`, `core/benchmark_test.go`, `core/endpoint_test.go`, `core/router_test.go`, `protocol/benchmark_test.go`, `protocol/fuzz_test.go`, `protocol/protocol_test.go`, `sdk/c/tests/CMakeLists.txt`, `sdk/c/tests/test_chat_json.c`, `sdk/c/tests/test_eipc_easy.c`, `sdk/c/tests/test_frame.c`, `sdk/c/tests/test_hmac.c`, and 39 more.

## Documented test commands

These commands are reproduced from the inspected root README or contributing guide:

```bash
make test           # go test -race ./...
```

```bash
go test -race -v ./...
```

```bash
cmake --build .
```

```bash
ctest --output-on-failure
```

## Verification baseline

This inventory comes from `master` at [`4cd7e5abac0a`](https://github.com/embeddedos-org/eIPC/commit/4cd7e5abac0a296d331f4c15de4e49ce00d3cf9a) and found 51 test-related paths among 194 files. Re-check the source tree when that commit is no longer current.
