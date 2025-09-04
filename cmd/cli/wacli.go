package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"

	wacli "github.com/marquesch/wasvc/internal/cli"
	"github.com/marquesch/wasvc/internal/socket"
	"github.com/urfave/cli/v3"
)

func main() {
	var phoneNumber string
	var body string
	var filePath string
	var caption string
	var follow bool
	var tail int8

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
								Args:       []any{phoneNumber, body},
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
								Args:       []any{phoneNumber, filePath, caption},
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
						Args:    []any{phoneNumber},
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
						Name:        "follow",
						Destination: &follow,
					},
					&cli.Int8Flag{
						Name:        "tail",
						Destination: &tail,
						Value:       20,
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					command := socket.ClientCommand{
						Command: "get",
						Args:    []any{phoneNumber, tail, follow},
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

					reader := bufio.NewReader(conn)
					err = socket.ReadEvent(reader, &response)
					if err != nil {
						return fmt.Errorf("read error: %w", err)
					}

					if !response.Success {
						return errors.New("server responded unsuccessfully")
					}

					eventChan := make(chan socket.ServerResponse)
					go func() {
						var evt socket.ServerResponse
						for {
							err := socket.ReadEvent(reader, &evt)
							if err != nil {
								return
							}

							eventChan <- evt
						}
					}()

					for {
						select {
						case evt := <-eventChan:
							if evt.Success {
								fmt.Fprint(os.Stdout, evt.Message)
							} else {
								return nil
							}
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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)

	if err := cmd.Run(ctx, os.Args); err != nil {
		stop()
		panic(err)
	}
}
