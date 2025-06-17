.PHONY: fmt proto build test

fmt:
	golines -w .
	gofumpt -w .

# Generate protobuf code
proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		internal/client/grpc/proto/gate.proto

# Build the application
build:
	go build -o bin/upbit-api-poll ./main.go

# Run tests
test:
	go test -v ./...