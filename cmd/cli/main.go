package main

import (
	"fmt"
	"net"
	"os"

	"github.com/marquesch/wasvc/internal/socket"
)

const socketPath = "/tmp/app.sock"

func main() {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		fmt.Println("Dial error:", err)
		os.Exit(1)
	}
	defer conn.Close()

	message := socket.SocketMessage{
		Message: "Olá",
		ErrStr:  "",
	}

	socket.WriteMessage(conn, message)

	response, err := socket.ReadMessage(conn)
	if err != nil {
		fmt.Println("read error: ", err)
		return
	}

	if response.ErrStr != "" {
		fmt.Println("response error: ", response.ErrStr)
		return
	}

	fmt.Printf("got succesful response: %s", response.Message)
}

// import (
// 	"context"
// 	"fmt"
// 	"os"
//
// 	_ "github.com/mattn/go-sqlite3"
// 	"github.com/mdp/qrterminal/v3"
// 	"github.com/urfave/cli/v3"
// 	"go.mau.fi/whatsmeow"
// 	"go.mau.fi/whatsmeow/proto/waE2E"
// 	"go.mau.fi/whatsmeow/store/sqlstore"
// 	"go.mau.fi/whatsmeow/types"
// 	waLog "go.mau.fi/whatsmeow/util/log"
// 	"google.golang.org/protobuf/proto"
// )
//
// func connect() (*whatsmeow.Client, error) {
// 	dbLog := waLog.Stdout("Database", "ERROR", true)
//
// 	ctx := context.Background()
// 	container, err := sqlstore.New(ctx, "sqlite3", "file:examplestore.db?_foreign_keys=on", dbLog)
// 	if err != nil {
// 		return nil, fmt.Errorf("error setting db: %w", err)
// 	}
//
// 	deviceStore, err := container.GetFirstDevice(ctx)
// 	if err != nil {
// 		return nil, fmt.Errorf("error setting device: %w", err)
// 	}
//
// 	clientLog := waLog.Stdout("Client", "ERROR", true)
// 	client := whatsmeow.NewClient(deviceStore, clientLog)
//
// 	if client.Store.ID == nil {
// 		qrChan, _ := client.GetQRChannel(context.Background())
//
// 		err = client.Connect()
// 		if err != nil {
// 			return nil, fmt.Errorf("error connecting to client: %w", err)
// 		}
//
// 		for evt := range qrChan {
// 			if evt.Event == "code" {
// 				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
// 			}
// 		}
//
// 	} else {
// 		err = client.Connect()
// 		if err != nil {
// 			return nil, fmt.Errorf("error connecting to client: %w", err)
// 		}
// 	}
//
// 	return client, nil
// }
//
// func sendTextMessage(client *whatsmeow.Client, recipient string, text string) error {
// 	toJID, _ := types.ParseJID(fmt.Sprintf("%s@s.whatsapp.net", recipient))
//
// 	message := &waE2E.Message{
// 		Conversation: proto.String(text),
// 	}
//
// 	_, err := client.SendMessage(context.Background(), toJID, message)
// 	if err != nil {
// 		return fmt.Errorf("error sending message: %w", err)
// 	}
//
// 	return nil
// }
//
// func main() {
// 	client, err := connect()
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	var recipient string
// 	var text string
//
// 	cmd := &cli.Command{
// 		Name:  "wacli",
// 		Usage: "whatsapp cli tool",
// 		Action: func(ctx context.Context, cmd *cli.Command) error {
// 			return nil
// 		},
// 		Commands: []*cli.Command{
// 			{
// 				Name: "send",
// 				Commands: []*cli.Command{
// 					{
// 						Name: "text",
// 						Arguments: []cli.Argument{
// 							&cli.StringArg{
// 								Name:        "recipient",
// 								Destination: &recipient,
// 							},
// 							&cli.StringArg{
// 								Name:        "text",
// 								Destination: &text,
// 							},
// 						},
// 						Action: func(ctx context.Context, cmd *cli.Command) error {
// 							fmt.Printf("inside action\ntext: %s\nrecipient: %s", text, recipient)
// 							err := sendTextMessage(client, recipient, text)
// 							return err
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}
//
// 	if err := cmd.Run(context.Background(), os.Args); err != nil {
// 		panic(err)
// 	}
// }
