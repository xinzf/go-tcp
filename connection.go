package tcp

import (
	"errors"
	// "fmt"
	"net"
	"sync"
	"sync/atomic"
)

var (
	ErrorWriteSize  = errors.New("The length of the data to write is not correct")
	ErrorReadClosed = errors.New("The read channel has been closed")
	ErrorSendClosed = errors.New("The write channel has been closed")
	ErrorWriteBlock = errors.New("The send channel is blocked")
)

func NewConnection(s *server, conn *net.TCPConn) *Connection {
	c := &Connection{
		server:      s,
		eventer:     s.protoManager.GetEventer(),
		reader:      s.protoManager.GetReader(),
		packeter:    s.protoManager.GetPacketer(),
		conn:        conn,
		receiveChan: make(chan Packeter, Options.ReceiveChanLimit),
		sendChan:    make(chan Packeter, Options.SendChanLimit),
	}

	c.Info.RemoteAddr = conn.RemoteAddr().String()
	c.Info.LocalAddr = conn.LocalAddr().String()
	c.Info.Extended = make(map[string]interface{})
	c.eventer.OnConnection(c)
	return c
}

type Connection struct {
	server      *server
	conn        *net.TCPConn
	receiveChan chan Packeter
	sendChan    chan Packeter
	closeOnce   sync.Once
	closeFlag   int32
	eventer     Eventer
	reader      Reader
	packeter    Packeter
	Info        struct {
		RemoteAddr string
		LocalAddr  string
		Extended   map[string]interface{}
	}
}

func (this *Connection) handleRead() {
	go func() {
		for {
			select {
			case p, ok := <-this.receiveChan:
				if ok == false {
					this.Close()
					return
				}

				this.eventer.OnMessage(this, p)
			}
		}
	}()
}

func (this *Connection) handleReceive() {
	go func() {
		for {
			packet, err := this.reader.Read(this.conn)
			if err != nil {
				this.Close()
				return
			}

			if packet.Size() > 0 {
				this.receiveChan <- packet
			}
		}
	}()
}

func (this *Connection) handleSend() {
	go func() {
		for {
			select {
			case p, ok := <-this.sendChan:
				if ok == false {
					this.Close()
					return
				}

				size, err := this.conn.Write(p.Serialize())
				if err != nil {
					this.Close()
					return
				}

				if size != p.Size() {
					this.Close()
					return
				}

				if p.Close() == true {
					this.Close()
					return
				}
			}
		}
	}()
}

func (this *Connection) monitor() {
	this.handleReceive()
	this.handleRead()
	this.handleSend()
}

func (this *Connection) Send(p Packeter) error {
	select {
	case this.sendChan <- p:
		return nil
	default:
		return ErrorWriteBlock
	}
}

func (this *Connection) Close() {
	if this.IsClosed() {
		return
	}

	this.closeOnce.Do(func() {
		atomic.StoreInt32(&this.closeFlag, 1)
		close(this.sendChan)
		close(this.receiveChan)
		this.conn.Close()
		this.eventer.OnClose(this)
	})
}

func (this *Connection) IsClosed() bool {
	return atomic.LoadInt32(&this.closeFlag) == 1
}
