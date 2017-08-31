package main

import (
	"context"
	"fmt"
	"github.com/xinzf/go-tcp"
	"github.com/xinzf/go-utils"
	"net"
	"strings"
)

var BufferPool *utils.ObjectPool

func init() {
	BufferPool = utils.NewObjectPool(10, 1024, func() interface{} {
		return make([]byte, 1024)
	})
}

type Callback struct{}

func (this *Callback) OnConnection(conn *tcp.Connection) error {
	conn.Info.Extended = map[string]interface{}{
		"id":   "conn id",
		"name": "user name",
	}
	// fmt.Println(conn.Info)
	return nil
}

func (this *Callback) OnMessage(conn *tcp.Connection, packet tcp.Packeter) {
	var m tcp.Packeter
	if packet.String() == "quit" {
		m = &Message{
			data:      packet.Serialize(),
			length:    packet.Size(),
			closeFlag: true,
		}
	} else {
		m = &Message{
			data:   packet.Serialize(),
			length: packet.Size(),
		}
	}
	conn.Send(m)
}

func (this *Callback) OnClose(conn *tcp.Connection) {
	fmt.Println("closed")
}

type Message struct {
	data      []byte
	length    int
	closeFlag bool
}

func (this *Message) SetData(data []byte) {
	this.data = data
	this.length = len(data)
}

func (this *Message) String() string {
	return strings.TrimSpace(string(this.data))
}

func (this *Message) Serialize() []byte {
	return this.data
}

func (this *Message) Size() int {
	return this.length
}

func (this *Message) Close() bool {
	return this.closeFlag
}

type Reader struct{}

func (this *Reader) Read(conn *net.TCPConn) (tcp.Packeter, error) {
	buf := BufferPool.Get().([]byte)
	size, err := conn.Read(buf)
	if err != nil {
		return new(Message), err
	}

	m := &Message{
		data:   buf[:size],
		length: size,
	}

	BufferPool.Put(buf)
	return m, nil
}

type Manager struct{}

func (this *Manager) GetEventer(args ...interface{}) tcp.Eventer {
	var c tcp.Eventer = new(Callback)
	return c
}

func (this *Manager) GetPacketer(args ...interface{}) tcp.Packeter {
	var p tcp.Packeter = new(Message)
	return p
}

func (this *Manager) GetReader(args ...interface{}) tcp.Reader {
	var r tcp.Reader = new(Reader)
	return r
}

func main() {
	tcp.Options.Addr = "127.0.0.1:8991"
	tcp.Options.AcceptConcurrent = 10
	tcp.Options.ReceiveChanLimit = 100
	tcp.Options.SendChanLimit = 100
	tcp.Options.Debug = true

	ctx, _ := context.WithCancel(context.Background())

	server := tcp.NewServer(ctx)
	server.SetProtocolManager(&Manager{})
	if err := server.Start(); err != nil {
		panic(err)
	}
}
