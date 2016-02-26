package rpc

type ResponseListener interface {
	ResponseReceived(term uint32)
}
