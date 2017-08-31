package tcp

var Options struct {
	Addr             string
	AcceptConcurrent int
	ReceiveChanLimit int
	SendChanLimit    int
}
