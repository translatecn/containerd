package ttrpc

type StreamServer interface {
	SendMsg(m interface{}) error
	RecvMsg(m interface{}) error
}
