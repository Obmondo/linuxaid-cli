package webtee

import (
	"context"
	"crypto/tls"
	"log/slog"
	"os"
	"time"

	rpc "gitea.obmondo.com/EnableIT/go-scripts/rpc"

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

	app.conn, err = grpc.NewClient(app.config.Server(), opts...)

	isConnected := false

	if err != nil {
		slog.Debug("failed to connect to webtee server", slog.String("error", err.Error()))
	} else {
		isConnected = true
	}

	// If config.ContinueOnDisconnect is true, we want to continue executing command,
	// even if connection to server failed.

	if !(isConnected) && !(app.config.ContinueOnDisconnect()) {
		slog.Debug("ContinueOnDisconnect is set to false so quitting without executing command")
		os.Exit(1)
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
		slog.Debug("failed to initialize log stream", slog.String("error", err.Error()))
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
				slog.Debug("failed to close log stream", slog.String("error", err.Error()))
			}

			return
		}

		logLine := rpc.LogLine{
			Timestamp: uint64(time.Now().Unix()),
			Line:      m.line,
			Pipe:      m.pipe,
		}

		if err := stream.Send(&logLine); err != nil {
			slog.Debug("failed to send", slog.String("log_line", logLine.String()), slog.String("error", err.Error()))
		}
	}

}
