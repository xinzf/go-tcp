package tcp

import (
	"context"
	"log"
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
	globalCtx  context.Context
	serverCtx  context.Context
	stopSignal context.CancelFunc
	onConnect  OnConnect
	onMessage  OnMessage
	onClose    OnClose
	packetRead PacketRead
	l          *net.TCPListener
	closeOnce  sync.Once
}

func (this *server) SetOnConnect(fun OnConnect) {
	this.onConnect = fun
}

func (this *server) SetOnMessage(fun OnMessage) {
	this.onMessage = fun
}

func (this *server) SetOnClose(fun OnClose) {
	this.onClose = fun
}

func (this *server) SetPacketRead(fun PacketRead) {
	this.packetRead = fun
}

func (this *server) Start() error {
	addr, err := net.ResolveTCPAddr("tcp4", Options.Addr)
	if err != nil {
		log.Println(err)
		return err
	}

	l, err := net.ListenTCP("tcp4", addr)
	if err != nil {
		log.Println(err)
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
			log.Println(err)
			return
		}

		conn := NewConnection(this, c)
		er := this.onConnect(conn)
		if er != nil {
			conn.Close()
		} else {
			conn.monitor()
		}
	}
}

func (this *server) Shutdown() {
	this.stopSignal()
}

func (this *server) close() {
	this.closeOnce.Do(func() {
		this.l.Close()
		if Options.Debug {
			log.Println("tcp server has shutdown")
		}
	})
}
