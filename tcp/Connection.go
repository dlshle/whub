package tcp

import "net"

const (
	DefaultReadBufferSize = 4096
)

type TCPConnection struct {
	conn        net.Conn
	onMessageCb func([]byte)
	onCloseCb   func(error)
	onErrorCb   func(error)
}

type ITCPConnection interface {
	Close() error
	Read() ([]byte, error)
	OnMessage(func([]byte))
	Write([]byte) error
	Address() string
	OnError(func(error))
	OnClose(func(error))
	State() int
	StartListening()
	ReadLoop()
	StopListening()
	String() string
}

func (c *TCPConnection) Close() error {
	err := c.conn.Close()
	c.handleClose(err)
	return err
}

func (c *TCPConnection) Read() ([]byte, error) {
	buffer := make([]byte, DefaultReadBufferSize)
	_, err := c.conn.Read(buffer)
	if err != nil {
		c.handleError(err)
	} else {
		c.handleMessage(buffer)
	}
	return buffer, err
}

func (c *TCPConnection) OnMessage(cb func([]byte)) {
	c.onMessageCb = cb
}

func (c *TCPConnection) handleMessage(message []byte) {
	if c.onMessageCb != nil {
		c.onMessageCb(message)
	}
}

func (c *TCPConnection) Write(data []byte) error {
	_, err := c.conn.Write(data)
	if err != nil {
		c.handleError(err)
	}
	return err
}

func (c *TCPConnection) Address() string {
	return c.conn.RemoteAddr().String()
}

func (c *TCPConnection) OnError(cb func(err error)) {
	c.onErrorCb = cb
}

func (c *TCPConnection) handleError(err error) {
	if c.onErrorCb != nil {
		c.onErrorCb(err)
	} else {
		c.Close()
	}
}

func (c *TCPConnection) OnClose(cb func(err error)) {
	c.onCloseCb = cb
}

func (c *TCPConnection) handleClose(err error) {
	if c.onCloseCb != nil {
		c.onCloseCb(err)
	}
}

func (c *TCPConnection) State() int {
	return 0
}

func (c *TCPConnection) StartListening() {
	go c.ReadLoop()
}

func (c *TCPConnection) ReadLoop() {
	// TODO
}

func (c *TCPConnection) StopListening() {
	// TODO
}

func (c *TCPConnection) String() string {
	return ""
}
