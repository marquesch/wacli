package socket

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
)

const SocketPath = "/tmp/app.sock"

type ISocketServer interface {
	Start() error
	Accept() (net.Conn, error)
}

type Connection struct {
	Context context.Context
	Conn    net.Conn
	bufio.Reader
	bufio.Writer
}

func (connection *Connection) ReadRequest() (*Request, string, error) {
	payload, err := connection.ReadString('\n')
	if err != nil {
		return nil, payload, err
	}

	var req Request
	err = json.Unmarshal([]byte(payload), &req)
	if err != nil {
		return nil, payload, err
	}

	return &req, payload, nil
}

func Write(conn net.Conn, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error marshaling json: %w", err)
	}

	_, err = conn.Write((append(payload, '\n')))
	if err != nil {
		return fmt.Errorf("error writing to channel: %w", err)
	}

	return nil
}

type SocketServer struct {
	SocketPath string
	listener   net.Listener
	handlers   map[string]RequestHandler
}

func (ss *SocketServer) Start() error {
	if ss.listener != nil {
		return errors.New("server has already been started")
	}

	err := os.RemoveAll(ss.SocketPath)
	if err != nil {
		return fmt.Errorf("error removing old socket file: %w", err)
	}

	ss.listener, err = net.Listen("unix", ss.SocketPath)
	if err != nil {
		return fmt.Errorf("error listening to socket: %w", err)
	}

	return nil
}

func (ss *SocketServer) Listen() {
	for {
		conn, err := ss.listener.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}

		go ss.HandleConnection(conn)
	}
}

func (ss *SocketServer) HandleConnection(conn net.Conn) {
	defer conn.Close()

	req, payload, err := ReadRequest(conn)
	if err != nil {
		Write(conn, Response{Status: "error"})
		return
	}

	handler, exists := ss.handlers[req.Command]
	if !exists {
		Write(conn, Response{
			Status:        "error",
			TransactionID: req.TransactionID,
			Error:         "unknown command",
			Data:          nil,
		})
		return
	}

	response := handler.Handle(*req, json.RawMessage(payload))
	Write(conn, response)
}

type SocketService struct {
	ISocketServer
	log.Logger
}

type RequestHandler interface {
	Handle(Request, json.RawMessage) Response
}

type Request struct {
	TransactionID string `json:"transaction_id"`
	Command       string `json:"command"`
}

type Response struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
	Data          any    `json:"data,omitempty"`
	Error         string `json:"error,omitempty"`
}
