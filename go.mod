module github.com/hacksomecn/protobuf-go

go 1.11

require (
	github.com/golang/protobuf v1.5.0
	github.com/google/go-cmp v0.5.5
	google.golang.org/protobuf v0.0.0
)

replace (
	google.golang.org/protobuf => github.com/hacksomecn/protobuf-go v1.1000.0
)