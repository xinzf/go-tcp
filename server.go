package tcp

import (
	"context"
	"net"
	"sync"
)

func NewServer(ctx context.Context) *server {
	myctx, canclefunc := context.WithCancel(context.Background())
	return &server{
		globalCtx:  ctx,
		serverCtx:  myctx,
		stopSignal: canclefunc,
	}
}

type server struct {
	globalCtx    context.Context
	serverCtx    context.Context
	stopSignal   context.CancelFunc
	protoManager ProtocolManager
	l            *net.TCPListener
	closeOnce    sync.Once
}

func (this *server) SetProtocolManager(m ProtocolManager) {
	this.protoManager = m
}

func (this *server) Start() error {
	addr, err := net.ResolveTCPAddr("tcp4", Options.Addr)
	if err != nil {
		return err
	}

	l, err := net.ListenTCP("tcp4", addr)
	if err != nil {
		return err
	}

	this.l = l

	for i := 0; i < Options.AcceptConcurrent; i++ {
		go this.accept()
	}

EXIT:
	for {
		select {
		case <-this.globalCtx.Done():
			this.Shutdown()
		case <-this.serverCtx.Done():
			this.close()
			break EXIT
		}
	}
	return nil
}

func (this *server) accept() {
	for {
		c, err := this.l.AcceptTCP()
		if err != nil {
			// logger
			return
		}

		conn := NewConnection(this, c)
		conn.monitor()
	}
}

func (this *server) Shutdown() {
	this.stopSignal()
}

func (this *server) close() {
	this.closeOnce.Do(func() {
		this.l.Close()
	})
}
