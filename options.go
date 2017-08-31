package tcp

var Options struct {
	Debug            bool
	Addr             string
	AcceptConcurrent int
	ReceiveChanLimit int
	SendChanLimit    int
}
