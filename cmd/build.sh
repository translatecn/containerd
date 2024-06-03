CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/ ./containerd
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/ ./containerd-shim-runc-v2
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/ ./containerd-stress
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/ ./ctr
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/ ./gen-manpages
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/ ./protoc-gen-go-fieldpath
