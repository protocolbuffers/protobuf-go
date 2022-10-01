# Fork of [Protobuf Golang](https://github.com/protocolbuffers/protobuf-go) specialized for type generation

This package supposed to focus on type generation based on proto file. 

## Features compared to original repository

- Flags to REMOVE protobuf specific field generation
- Make difference between optional and non-optional in case of structs. So it's generating non-pointer in case of non-optional

## Installation

```shell
go install github.com/infiniteloopcloud/protoc-gen-go-types@latest
```

## Flags

This are actually environment variables.

- `SKIP_VERSION_MARKERS=false` - Skip markers
- `SKIP_EXTENSIONS=false` - Skip extensions
- `SKIP_REFLECT_FILE_DESCRIPTOR=false` - Skip reflect file descriptors
