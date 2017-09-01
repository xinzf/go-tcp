package tcp

import (
	"net"
)

type OnConnect func(conn *Connection) error

type OnMessage func(conn *Connection, packet Packet)

type OnClose func(conn *Connection)

type PacketRead func(conn *net.TCPConn) (Packet, error)

type Packet struct {
	Data  []byte
	Size  int
	Close bool
}
