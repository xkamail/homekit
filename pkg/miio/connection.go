package miio

import (
	"encoding/json"
	"log/slog"
	"net"
)

// Gateway command.
type command struct {
	Sid   string `json:"sid,omitempty"`
	Model string `json:"model,omitempty"`
	Data  string `json:"data,omitempty"`
	Token string `json:"token,omitempty"`
	Cmd   string `json:"cmd"`
}

// Base connection.
type connection struct {
	conn *net.UDPConn

	closeRead  chan bool
	closeWrite chan bool

	inMessages  chan []byte
	outMessages chan []byte

	DeviceMessages chan []byte
}

// Creates a new connection.
func newConnection(ip string, port int) (*connection, error) {
	addr := &net.UDPAddr{
		IP:   net.ParseIP(ip),
		Port: port,
	}

	con, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		return nil, err
	}

	c := &connection{
		conn:           con,
		closeWrite:     make(chan bool),
		closeRead:      make(chan bool),
		inMessages:     make(chan []byte),
		outMessages:    make(chan []byte),
		DeviceMessages: make(chan []byte),
	}

	c.start()
	return c, nil
}

// Close closes the connection.
func (c *connection) Close() {
	if nil != c.conn {
		c.conn.Close()
	}

	c.closeRead <- true
	c.closeWrite <- true

	close(c.inMessages)
	close(c.outMessages)
	close(c.closeRead)
	close(c.closeWrite)
	close(c.DeviceMessages)
}

// Send sends a new message.
func (c *connection) Send(cmd *command) error {
	out, err := json.Marshal(cmd)
	if err != nil {
		return err
	}
	c.outMessages <- out
	return nil
}

// Starts the listeners.
func (c *connection) start() {
	go c.in()
	go c.out()
}

// Processes incoming messages.
func (c *connection) in() {
	buf := make([]byte, 2048)
	for {
		select {
		case <-c.closeRead:
			return
		default:
			size, _, err := c.conn.ReadFromUDP(buf)
			if err != nil {
				slog.Error("Failed to read from UDP", "error", err)
				continue
			}

			if size > 0 {
				msg := make([]byte, size)
				copy(msg, buf[0:size])
				c.DeviceMessages <- msg
			}
		}
	}
}

// Processes outgoing messages.
func (c *connection) out() {
	for {
		select {
		case <-c.closeWrite:
			return
		case msg, ok := <-c.outMessages:
			if !ok {
				return
			}

			if _, err := c.conn.Write(msg); err != nil {
				slog.Error("Failed to write to UDP", "error", err)
			}
		}
	}
}
