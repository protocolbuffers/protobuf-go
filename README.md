## Fork of [Protobuf Golang](https://github.com/protocolbuffers/protobuf-go) specialized for type generation

This package supposed to focus on type generation based on proto file. 

### Features compared to original repository

- Flags to REMOVE protobuf specific field generation
- Make difference between optional and non-optional in case of structs. So it's generating non-pointer in case of non-optional

### Installation

```shell
go install github.com/infiniteloopcloud/protoc-gen-go-types@latest
```

### Flags

These are actually environment variables.

- `SKIP_PROTOBUF_SPECIFIC=false` - Skip protobuf specific code
- `TYPE_OVERRIDE` - Enable type override

### Type override

Supported overwrite:

- `TimeTime` -> `time.Time`

#### Example

```protobuf
syntax = "proto3";

message SomeStruct {
  TimeTime created_at = 1; // This will be time.Time
}

message TimeTime {}
```

#### Important

Currently the generator not importing the `time` package automatically. Temporary solution can be: `goimports -w *.pb.go`. 
