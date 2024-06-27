CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/ ./cmd/containerd
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/ ./cmd/containerd-shim-runc-v2
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/ ./cmd/ctd-decoder
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/ ./cmd/ctr
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/ ./third_party/github.com/kubernetes-sigs/cri-tools/cmd/crictl

#rm -rf bin