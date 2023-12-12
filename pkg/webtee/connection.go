package webtee

import (
	"context"
	"crypto/tls"
	"log"
	"time"

	rpc "go-scripts/rpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

const metadataCertKey = "enableit-cert"

type logLine struct {
	line string
	pipe string
}

func connectToServer(app *application) {
	// Initialize connection to webtee server
	var err error
	opts := []grpc.DialOption{
		getTLSDialOption(app.config.NoTLS()),
	}
	app.conn, err = grpc.Dial(app.config.Server(), opts...)

	isConnected := false

	if err != nil {
		log.Printf("Failed to connect to webtee server: %v\n", err)
	} else {
		isConnected = true
	}

	// If config.ContinueOnDisconnect is true, we want to continue executing command,
	// even if connection to server failed.

	if !(isConnected) && !(app.config.ContinueOnDisconnect()) {
		log.Fatalln("ContinueOnDisconnect is set to false so quitting without executing command")
	}
}

func getTLSDialOption(noTLS bool) grpc.DialOption {
	if noTLS {
		return grpc.WithTransportCredentials(insecure.NewCredentials())
	}
	// since we did not provide root CAs, the lib will use the OS's set
	return grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{}))
}

// It should always be run in a separate goroutine because
// we decrement  app.wg waitgroup after execution.
func webTee(app *application, lines <-chan logLine) {
	defer app.wg.Done()

	client := rpc.NewWebteeClient(app.conn)

	// Create new context with metadata for this stream.
	ctx := metadata.AppendToOutgoingContext(context.Background(), metadataCertKey, app.config.Cert())

	// Start the logging stream.
	stream, err := client.SendLog(ctx)
	if err != nil {
		log.Printf("Failed to initialize log stream: %v\n", err)
		// Since we can't connect to server, accept and discard whatever we receive in the lines channel.
		for {
			_, more := <-lines

			if !more {
				return
			}
		}
	}

	for {
		m, more := <-lines

		if !more {
			_, err := stream.CloseAndRecv()
			// fail here, when cert is not found in db
			if err != nil {
				log.Printf("Failed to close log stream: %v\n", err)
			}

			return
		}

		logLine := rpc.LogLine{
			Timestamp: uint64(time.Now().Unix()),
			Line:      m.line,
			Pipe:      m.pipe,
		}

		if err := stream.Send(&logLine); err != nil {
			log.Printf("Failed to send log line (%v): %v\n", logLine.String(), err)
		}
	}

}
