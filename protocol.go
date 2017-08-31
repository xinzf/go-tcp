package tcp

import (
	"net"
)

type ProtocolManager interface {
	GetEventer(args ...interface{}) Eventer
	GetPacketer(args ...interface{}) Packeter
	GetReader(args ...interface{}) Reader
}

type Eventer interface {
	OnConnection(conn *Connection) error
	OnMessage(conn *Connection, packet Packeter)
	OnClose(conn *Connection)
}

type Packeter interface {
	Serialize() []byte
	String() string
	Size() int
	Close() bool
}

type Reader interface {
	Read(conn *net.TCPConn) (Packeter, error)
}
