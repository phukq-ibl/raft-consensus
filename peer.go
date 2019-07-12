package mynetwork

import (
	"bufio"
	"fmt"
	"net"
)

type Peer struct {
	Id   string
	conn net.Conn
}

type HandleFunc func(*Peer) error

func NewPeer(conn net.Conn) *Peer {
	return &Peer{Id: "", conn: conn}
}

func (p Peer) Send(msg string) {
	fmt.Fprintln(p.conn, msg)
}

func (p Peer) HandShake() {

}
func (p Peer) Run(hf HandleFunc) error {
	err := hf(&p)
	return err
}

func (p Peer) ReadMsg() (string, error) {
	msg, err := bufio.NewReader(p.conn).ReadString('\n')
	return msg, err
}

func (p Peer) SendMsg(msg string) error {
	_, err := fmt.Fprint(p.conn, msg+"\n")
	return err
}

func (p Peer) Disconnect() error {
	return p.conn.Close()

}

func (p Peer) RemoteAddr() net.Addr {
	return p.conn.RemoteAddr()
}
