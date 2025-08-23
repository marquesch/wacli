package main

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/marquesch/wasvc/internal/socket"
	"github.com/urfave/cli/v3"
)

func sendCommand(clientEvent socket.ClientEvent) (socket.ServerEvent, error) {
	conn, err := net.Dial("unix", socket.SocketPath)
	if err != nil {
		fmt.Println("Dial error:", err)
		os.Exit(1)
	}
	defer conn.Close()

	err = socket.WriteEvent(conn, clientEvent)
	if err != nil {
		fmt.Println("write error: ", err)
		os.Exit(1)
	}

	var response socket.ServerEvent

	err = socket.ReadEvent(conn, &response)
	if err != nil {
		fmt.Println("read error: ", err)
		os.Exit(1)
	}

	return response, nil
}

func main() {
	var phoneNumber string
	var body string

	cmd := &cli.Command{
		Name:  "wacli",
		Usage: "whatsapp cli tool",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return nil
		},
		Commands: []*cli.Command{
			{
				Name: "send",
				Commands: []*cli.Command{
					{
						Name: "text",
						Arguments: []cli.Argument{
							&cli.StringArg{
								Name:        "recipient",
								Destination: &phoneNumber,
							},
							&cli.StringArg{
								Name:        "body",
								Destination: &body,
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							message := socket.ClientEvent{
								Command: "send",
								Args:    []string{phoneNumber, body},
							}

							response, err := sendCommand(message)
							if err != nil {
								return fmt.Errorf("error sending action to server: %w", err)
							}

							if !response.Success {
								fmt.Println(response.Error)
							}

							return nil
						},
					},
				},
			},
			{
				Name: "check",
				Arguments: []cli.Argument{
					&cli.StringArg{
						Name:        "phoneNumber",
						Destination: &phoneNumber,
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					event := socket.ClientEvent{
						Command: "check",
						Args:    []string{phoneNumber},
					}

					response, err := sendCommand(event)
					if err != nil {
						return fmt.Errorf("error sending command: %w", err)
					}

					if response.Error != "" {
						return fmt.Errorf("server error: %w", err)
					}

					fmt.Println(response.Success)
					return nil
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		panic(err)
	}
}
