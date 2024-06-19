package proto

//go:generate protoc --go_out=. manifest.proto
//go:generate mv demo/others/continuity/proto/manifest.pb.go .
//go:generate rmdir -p demo/others/continuity/proto
