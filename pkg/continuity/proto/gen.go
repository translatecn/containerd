package proto

//go:generate protoc --go_out=. manifest.proto
//go:generate mv demo/pkg/continuity/proto/manifest.pb.go .
//go:generate rmdir -p demo/pkg/continuity/proto
