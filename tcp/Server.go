package tcp

import (
	"context"
	"fmt"
	"net"
	"wsdk/common/connection"
	"wsdk/common/logger"
)

type TCPServer struct {
	name     string
	address  string
	port     int
	logger   *logger.SimpleLogger
	ctx      context.Context
	stopFunc func()

	onConnected     func(conn connection.IConnection)
	onDisconnected  func(conn connection.IConnection, err error)
	onConnectionErr func(conn connection.IConnection, err error)
}

func (s *TCPServer) Start() error {
	s.logger.Println("starting TCP server...")
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.address, s.port))
	if err != nil {
		return err
	}
	select {
	case <-s.ctx.Done():
		s.logger.Println("stopping server ...")
		break
	default:
		conn, err := listener.Accept()
		if err != nil {
			s.logger.Println("listener err: ", err)
			return err
		}
		go s.handleNewConnection(conn)
	}
	return listener.Close()
}

func (s *TCPServer) toTCPConnection(conn net.Conn) connection.IConnection {
	return NewTCPConnection(conn)
}

func (s *TCPServer) handleNewConnection(rawConn net.Conn) {
	conn := s.toTCPConnection(rawConn)
	s.logger.Println("new tcp connection ", conn.String())
	conn.OnError(func(err error) {
		s.onConnectionErr(conn, err)
	})
	conn.OnClose(func(err error) {
		s.onConnectionErr(conn, err)
	})
	s.onConnected(conn)
}

func (s *TCPServer) Stop() error {
	s.stopFunc()
	return nil
}

func (s *TCPServer) OnConnectionError(cb func(connection.IConnection, error)) {
	s.onConnectionErr = cb
}

func (s *TCPServer) OnClientConnected(cb func(connection.IConnection)) {
	s.onConnected = cb
}

func (s *TCPServer) OnClientClosed(cb func(connection.IConnection, error)) {
	s.onDisconnected = cb
}

func (s *TCPServer) SetLogger(logger *logger.SimpleLogger) {
	s.logger = logger
}
