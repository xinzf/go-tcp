package tcp

import (
	"errors"
	"fmt"
	"log"
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
		conn:        conn,
		receiveChan: make(chan Packet, Options.ReceiveChanLimit),
		sendChan:    make(chan Packet, Options.SendChanLimit),
	}

	c.Info.RemoteAddr = conn.RemoteAddr().String()
	c.Info.LocalAddr = conn.LocalAddr().String()
	c.Info.Extended = make(map[string]interface{})
	if Options.Debug {
		log.Println(fmt.Sprintf("new client:%s connected", c.Info.RemoteAddr))
	}
	return c
}

type Connection struct {
	server      *server
	conn        *net.TCPConn
	receiveChan chan Packet
	sendChan    chan Packet
	closeOnce   sync.Once
	closeFlag   int32
	Info        struct {
		RemoteAddr string
		LocalAddr  string
		Extended   map[string]interface{}
	}
}

func (this *Connection) handleRead() {
	go func() {
		defer func() {
			if Options.Debug {
				log.Println("read handle goroutine has exit")
			}
		}()

		for {
			select {
			case p, ok := <-this.receiveChan:
				if ok == false {
					this.Close()
					return
				}

				this.server.onMessage(this, p)
			}
		}
	}()
}

func (this *Connection) handleReceive() {
	go func() {
		defer func() {
			if Options.Debug {
				log.Println("receive handle goroutine has exit")
			}
		}()

		for {
			packet, err := this.server.packetRead(this.conn)
			if err != nil {
				this.Close()
				return
			}

			if packet.Size > 0 {
				this.receiveChan <- packet
			}
		}
	}()
}

func (this *Connection) handleSend() {
	go func() {
		defer func() {
			if Options.Debug {
				log.Println("send handle goroutine has exit")
			}
		}()

		for {
			select {
			case p, ok := <-this.sendChan:
				if ok == false {
					this.Close()
					return
				}

				size, err := this.conn.Write(p.Data)
				if err != nil {
					this.Close()
					return
				}

				if size != p.Size {
					this.Close()
					return
				}

				if p.Close == true {
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
	go func() {
		for {
			select {
			case <-this.server.serverCtx.Done():
				this.Close()
				return
			}
		}
	}()
}

func (this *Connection) Send(p Packet) error {
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
		if Options.Debug {
			log.Println(fmt.Sprintf("client:%s has closed", this.Info.RemoteAddr))
		}
		close(this.sendChan)
		close(this.receiveChan)
		this.conn.Close()
		this.server.onClose(this)
	})
}

func (this *Connection) IsClosed() bool {
	return atomic.LoadInt32(&this.closeFlag) == 1
}
