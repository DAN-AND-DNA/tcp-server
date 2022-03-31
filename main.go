package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"tcp-server/pkg/easyjson"
	"tcp-server/pkg/jsonrpc"
)

func main() {
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt)

	conf := easyjson.EasyConf{
		Address: "0.0.0.0:3000",
		HandleInput: func(conn *easyjson.EasyTcpConn, InputBuffer []byte) (Readed int, bytesResult []byte, err error) {
			err = checkBufferSize(InputBuffer, 2000)
			if err != nil {
				return 0, nil, err
			}

			return jsonrpc.Service(conn, InputBuffer)

		},

		HandleOutput: func(conn *easyjson.EasyTcpConn, OutputBuffer []byte) (err error) {
			checkBufferSize(OutputBuffer, 2000)

			return nil
		},
	}

	server := easyjson.NewEasyTcpServer()
	server.Run(conf)
	defer server.Stop()

	_ = <-stopChan
	log.Println("Aborting...")

}

func checkBufferSize(buffer []byte, maxSize int) error {
	if len(buffer) > maxSize {
		return fmt.Errorf("buffer reach max size")
	}

	return nil
}
