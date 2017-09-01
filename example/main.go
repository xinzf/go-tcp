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

var (
	connect tcp.OnConnect = func(conn *tcp.Connection) error {
		fmt.Println(conn.Info)
		return nil
	}

	receive tcp.OnMessage = func(conn *tcp.Connection, packet tcp.Packet) {
		conn.Send(packet)
		str := strings.TrimSpace(string(packet.Data))
		if str == "quit" {
			conn.Close()
		}
	}

	closed tcp.OnClose = func(conn *tcp.Connection) {
		fmt.Println(fmt.Sprintf("client: %s has closed", conn.Info.RemoteAddr))
	}

	reader tcp.PacketRead = func(conn *net.TCPConn) (p tcp.Packet, err error) {
		buf := BufferPool.Get().([]byte)
		size, err := conn.Read(buf)
		if err != nil {
			return
		}
		p = tcp.Packet{
			Data: buf[:size],
			Size: size,
		}
		BufferPool.Put(buf)
		return
	}
)

func main() {
	tcp.Options.Addr = "127.0.0.1:8991"
	tcp.Options.AcceptConcurrent = 10
	tcp.Options.ReceiveChanLimit = 100
	tcp.Options.SendChanLimit = 100
	tcp.Options.Debug = true

	ctx, _ := context.WithCancel(context.Background())

	server := tcp.NewServer(ctx)
	server.SetOnConnect(connect)
	server.SetOnMessage(receive)
	server.SetOnClose(closed)
	server.SetPacketRead(reader)
	if err := server.Start(); err != nil {
		panic(err)
	}
}
