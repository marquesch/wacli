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
	"sync"
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

type ClientConnection struct {
	Context    context.Context
	Conn       net.Conn
	writeChann chan Response
	mu         sync.Mutex
	bufio.Reader
}

func (connection *ClientConnection) ReadRequest() (*Request, string, error) {
	connection.mu.Lock()
	defer connection.mu.Unlock()

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

func (connection *ClientConnection) Write(data any) error {
	connection.mu.Lock()
	defer connection.mu.Unlock()

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = connection.Conn.Write(append(payload, '\n'))
	if err != nil {
		return err
	}

	return nil
}

func NewConnection(conn net.Conn) *ClientConnection {
	return &ClientConnection{
		Context:    context.Background(),
		Conn:       conn,
		writeChann: make(chan Response),
		mu:         sync.Mutex{},
		Reader:     *bufio.NewReader(conn),
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
	defer connection.Conn.Close()

	req, payload, err := connection.ReadRequest()
	if err != nil {
		if err == io.EOF {
			return
		}
		connection.Write(Response{Status: "error"})
		return
	}

	handler, exists := ss.handlers[req.Command]
	if !exists {
		connection.Write(Response{
			Status:        "error",
			TransactionID: req.TransactionID,
			Error:         "unknown command",
			Data:          nil,
		})
		return
	}

	response, eventChannel := handler.Handle(*req, json.RawMessage(payload))
	connection.Write(response)

	if eventChannel == nil {
		return
	}

	for event := range eventChannel {
		connection.Write(event)
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
