package main

import (
	"log"
	"net"
	"os"
	"testing"

	"crypto/rand"
	"crypto/rsa"

	"golang.org/x/crypto/ssh"
)

// Start a simple SSH server for testing
func startTestSSHServer(port string, username, password string) {
	config := &ssh.ServerConfig{
		NoClientAuth: false,
	}
	config.AddHostKey(generateHostKey())

	config.PasswordCallback = func(conn ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
		if conn.User() == username && string(pass) == password {
			return nil, nil
		}
		return nil, nil
	}

	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen for connection: %v", err)
	}
	// defer listener.Close()

	log.Printf("SSH server listening on %s", port)

	go func() {
		for {
			tcpConn, err := listener.Accept()
			if err != nil {
				log.Fatalf("failed to accept connection: %v", err)
			}
			go handleConnection(tcpConn, config)
		}
	}()
}

// Helper function to generate a host key
func generateHostKey() ssh.Signer {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("failed to generate host key: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		log.Fatalf("failed to create signer: %v", err)
	}
	return signer
}

// Handle SSH connections
func handleConnection(conn net.Conn, config *ssh.ServerConfig) {
	defer conn.Close()

	sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		log.Printf("failed to establish SSH connection: %v", err)
		return
	}
	log.Printf("logged in: %s", sshConn.User())

	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		go handleChannel(newChannel)
	}
}

func handleChannel(newChannel ssh.NewChannel) {
	channel, _, err := newChannel.Accept()
	if err != nil {
		log.Printf("could not accept channel: %v", err)
		return
	}
	defer channel.Close()

	// Simple echo server
	for {
		buf := make([]byte, 1024)
		n, err := channel.Read(buf)
		if err != nil {
			log.Printf("failed to read from channel: %v", err)
			return
		}
		// Echo back the received data
		if _, err := channel.Write(buf[:n]); err != nil {
			log.Printf("failed to write to channel: %v", err)
			return
		}
	}
}

// Test for readCSV function
func TestReadCSV(t *testing.T) {
	// Start the test SSH server
	startTestSSHServer(":8122", "testuserfromcsv", "testpass")

	// This test assumes you have a service running on localhost:8122 (SSH)
	//if !isPortOpen("127.0.0.1", "8122") {
	//	t.Errorf("Expected port 8122 to be open on localhost")
	//}

	// Create a temporary CSV file for testing
	tempFile, err := os.CreateTemp(t.TempDir(), "test.csv")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name()) // Clean up

	// Write test data to the CSV file
	_, err = tempFile.WriteString("Hostname,Port,Username,Password\n")
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	_, err = tempFile.WriteString("localhost,8122,testuserfromcsv,testpass\n")
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	// Read the CSV file
	nodes, err := readCSV(tempFile.Name())
	if err != nil {
		t.Fatalf("Error reading CSV: %v", err)
	}

	// Check if the data is read correctly
	if len(nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Hostname != "localhost" {
		t.Errorf("Expected hostname 'localhost', got '%s'", nodes[0].Hostname)
	}
	if nodes[0].Port != "8122" {
		t.Errorf("Expected port '8122', got '%s'", nodes[0].Port)
	}
	if nodes[0].Username != "testuserfromcsv" {
		t.Errorf("Expected username 'testuser', got '%s'", nodes[0].Username)
	}
	if nodes[0].Password != "testpass" {
		t.Errorf("Expected password 'testpass', got '%s'", nodes[0].Password)
	}

	// Check SSH Auth for the node
	for _, node := range nodes[:1] {
		if !checkSSHAuth(node) {
			t.Errorf("Expected SSH Auth to succeed for %s", node.Hostname)
		}
	}
}

func TestCLIInputSSHAuth(t *testing.T) {
	// Start the test SSH server
	startTestSSHServer(":8123", "testuser", "testpass")

	// This test assumes you have a service running on localhost:8123 (SSH)
	//if !isPortOpen("127.0.0.1", "8123") {
	//	t.Errorf("Expected port 8123 to be open on localhost")
	//}

	// Prepare a Node for testing
	node := Node{
		Hostname: "localhost",
		Port:     "8123",
		Username: "testuser",
		Password: "testpass",
	}

	// Check SSH Auth for the node
	if !checkSSHAuth(node) {
		t.Errorf("Expected SSH Auth to succeed for %s", node.Hostname)
	}
}
