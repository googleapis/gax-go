# Internal Test Protos

The `custom_error.proto` defines a custom error message used in unit tests for
`apierror.ErrDetails.ExtractProtoMessage`. This package is for internal testing
only and should not be used in any other context.

## Regeneration

To regenerate the protobuf Go code you will need the following:

* The protobuf compiler [protoc]
* The Go [protobuf plugin]
* The [goimports] tool

From this directory run the following command:
```sh
protoc -I. --go_out=. --go_opt=module=github.com/googleapis/gax-go/v2/apierror/internal/proto custom_error.proto
goimports -w .
```

Note: the `module` plugin option ensures the generated code is placed in this
directory, and not in several nested directories defined by `go_package` option.

[protoc]: https://github.com/protocolbuffers/protobuf#protocol-compiler-installation
[protobuf plugin]: https://developers.google.com/protocol-buffers/docs/reference/go-generated
[goimports]: https://pkg.go.dev/golang.org/x/tools/cmd/goimports