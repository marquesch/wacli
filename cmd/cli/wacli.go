package main

import (
	"context"
	"fmt"
	"os"

	wacli "github.com/marquesch/wasvc/internal/cli"
	"github.com/marquesch/wasvc/internal/socket"
	"github.com/urfave/cli/v3"
)

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
							event := socket.ClientEvent{
								Command: "send",
								Args:    []string{phoneNumber, body},
							}

							response, err := wacli.SendCommand(event)
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
					event := socket.ClientEvent{
						Command: "check",
						Args:    []string{phoneNumber},
					}

					response, err := wacli.SendCommand(event)
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
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		panic(err)
	}
}
