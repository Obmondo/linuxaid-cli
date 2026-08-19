package webtee

import (
	"sync"

	"google.golang.org/grpc"
)

// application struct to hold dependencies
type application struct {
	config WebTeeConfig
	conn   *grpc.ClientConn
	wg     sync.WaitGroup
}

// nolint: revive
type WebTeeConfig struct {
	server               string   // Server Address
	continueOnDisconnect bool     // Continue executing command even if connection failed
	command              []string // The program/command whose logs we want to stream
	cert                 string   // Puppet certname which uniquely identifies a machine
	noTLS                bool     // Don't use TLS to connect to server
}

func (c WebTeeConfig) Server() string {
	return c.server
}

func (c WebTeeConfig) ContinueOnDisconnect() bool {
	return c.continueOnDisconnect
}

func (c WebTeeConfig) Command() []string {
	return c.command
}

func (c WebTeeConfig) Cert() string {
	return c.cert
}

func (c WebTeeConfig) NoTLS() bool {
	return c.noTLS
}

type Config interface {
	Server() string
	ContinueOnDisconnect() bool
	Command() []string
	Cert() string
	NoTLS() bool
}
