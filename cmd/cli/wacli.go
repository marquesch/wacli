package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"

	wacli "github.com/marquesch/wasvc/internal/cli"
	"github.com/marquesch/wasvc/internal/socket"
	"github.com/urfave/cli/v3"
)

func main() {
	var phoneNumber string
	var body string
	var filePath string
	var caption string
	var showTimestamp bool

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
							command := socket.ClientCommand{
								Command:    "send",
								Subcommand: "text",
								Args:       []string{phoneNumber, body},
							}

							response, err := wacli.SendCommand(command)
							if err != nil {
								return fmt.Errorf("error sending command to server: %w", err)
							}

							if !response.Success {
								fmt.Println(response.Message)
							}

							return nil
						},
					},
					{
						Name: "media",
						Arguments: []cli.Argument{
							&cli.StringArg{
								Name:        "phone-number",
								Destination: &phoneNumber,
							},
							&cli.StringArg{
								Name:        "file-path",
								Destination: &filePath,
							},
						},
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:        "caption",
								Destination: &caption,
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							command := socket.ClientCommand{
								Command:    "send",
								Subcommand: "media",
								Args:       []string{phoneNumber, filePath, caption},
							}

							response, err := wacli.SendCommand(command)
							if err != nil {
								return fmt.Errorf("error sending command to server: %w", err)
							}

							if !response.Success {
								fmt.Println(response.Message)
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
					command := socket.ClientCommand{
						Command: "check",
						Args:    []string{phoneNumber},
					}

					response, err := wacli.SendCommand(command)
					if err != nil {
						return fmt.Errorf("error sending command to server: %w", err)
					}

					if !response.Success {
						return fmt.Errorf("server error: %s", response.Message)
					}

					fmt.Println(response.Message)
					return nil
				},
			},
			{
				Name: "get",
				Arguments: []cli.Argument{
					&cli.StringArg{
						Name:        "phone-number",
						Destination: &phoneNumber,
					},
				},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:        "show-timestamp",
						Destination: &showTimestamp,
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					command := socket.ClientCommand{
						Command: "get",
						Args:    []string{phoneNumber},
					}

					conn, err := net.Dial("unix", socket.SocketPath)
					if err != nil {
						return fmt.Errorf("error dialing server: %w", err)
					}
					defer conn.Close()

					err = socket.WriteEvent(conn, command)
					if err != nil {
						return fmt.Errorf("write error: %w", err)
					}

					var response socket.ServerResponse

					err = socket.ReadEvent(conn, &response)
					if err != nil {
						return fmt.Errorf("read error: %w", err)
					}

					if !response.Success {
						return errors.New("server responded unsuccessfully")
					}

					eventChan := make(chan socket.MessageReceivedEvent)
					go func() {
						var msg socket.MessageReceivedEvent
						for {
							err := socket.ReadEvent(conn, &msg)
							if err != nil {
								return
							}

							eventChan <- msg
						}
					}()

					for {
						select {
						case msg := <-eventChan:
							fmt.Println(msg.Body)
						case <-ctx.Done():
							command := socket.ClientCommand{Command: "cancel"}
							socket.WriteEvent(conn, command)
							return nil
						}
					}
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		panic(err)
	}
}
