package socket

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

const SocketPath = "/tmp/app.sock"

type ISocketServer interface {
	Start() error
	Accept() (net.Conn, error)
}

type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type Connection struct {
	Context context.Context
	net.Conn
	bufio.Reader
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

func (connection *Connection) WriteData(data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = connection.Write(append(payload, '\n'))
	if err != nil {
		return err
	}

	return nil
}

func NewConnection(conn net.Conn) *Connection {
	return &Connection{
		Context: context.Background(),
		Conn:    conn,
		Reader:  *bufio.NewReader(conn),
	}
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
	connection := NewConnection(conn)
	defer connection.Close()

	req, payload, err := connection.ReadRequest()
	if err != nil {
		if err == io.EOF {
			return
		}
		connection.WriteData(Response{Status: "error"})
		return
	}

	handler, exists := ss.handlers[req.Command]
	if !exists {
		connection.WriteData(Response{
			Status:        "error",
			TransactionID: req.TransactionID,
			Error:         "unknown command",
			Data:          nil,
		})
		return
	}

	response, eventChannel := handler.Handle(*req, json.RawMessage(payload))
	connection.WriteData(response)

	if eventChannel == nil {
		return
	}

	for event := range eventChannel {
		connection.WriteData(event)
	}
}

type SocketService struct {
	ISocketServer
	log.Logger
}

type RequestHandler interface {
	Handle(Request, json.RawMessage) (Response, chan Event)
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
