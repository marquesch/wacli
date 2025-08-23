package main

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/marquesch/wasvc/internal/socket"
	"github.com/urfave/cli/v3"
)

const socketPath = "/tmp/app.sock"

func main() {
	var recipient string
	var text string

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
								Destination: &recipient,
							},
							&cli.StringArg{
								Name:        "text",
								Destination: &text,
							},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							conn, err := net.Dial("unix", socketPath)
							if err != nil {
								fmt.Println("Dial error:", err)
								os.Exit(1)
							}
							defer conn.Close()

							message := socket.ClientEvent{
								Command: "send",
								Args:    []string{recipient, text},
							}

							err = socket.WriteEvent(conn, message)
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

							return nil
						},
					},
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		panic(err)
	}
}
