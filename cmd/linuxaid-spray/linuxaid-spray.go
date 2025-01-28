package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

type Node struct {
	Hostname string
	Port     string
	Username string
	Password string
}

var (
	inputCSV  string
	outputCSV string
	hostname  string
	port      string
	username  string
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "ssh-checker",
		Short: "Check SSH authentication for nodes",
		Run:   run,
	}

	rootCmd.Flags().StringVarP(&inputCSV, "input-csv", "i", "", "Input CSV file with nodes")
	rootCmd.Flags().StringVarP(&outputCSV, "output-csv", "o", "", "Output CSV file for results (required if input-csv is provided)")
	rootCmd.Flags().StringVarP(&hostname, "hostname", "H", "", "Hostname of the node")
	rootCmd.Flags().StringVarP(&port, "port", "p", "22", "Port (default is 22)")
	rootCmd.Flags().StringVarP(&username, "username", "u", "", "Username for SSH")

	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Error executing command: %v", err)
	}
}

func run(cmd *cobra.Command, args []string) {
	if inputCSV == "" && (hostname == "" || username == "") {
		log.Fatal("Either --input-csv must be provided or all of --hostname and --username must be specified.")
	}

	var nodes []Node

	if inputCSV != "" {
		if outputCSV == "" {
			log.Fatal("--output-csv must be specified if --input-csv is provided.")
		}
		var err error
		nodes, err = readCSV(inputCSV)
		if err != nil {
			log.Fatalf("Error reading CSV: %v", err)
		}
	} else {
		// Parse the hostname to extract user and hostname
		if strings.Contains(hostname, "@") {
			parts := strings.SplitN(hostname, "@", 2)
			username = parts[0]
			hostname = parts[1]
		}

		nodes = append(nodes, Node{
			Hostname: hostname,
			Port:     port,
			Username: username,
		})
	}

	checkNodes(nodes, outputCSV)
}

func readCSV(file string) ([]Node, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var nodes []Node
	for _, record := range records[1:] { // Skip header
		port := record[1]
		if port == "" {
			port = "22" // Default port
		}
		nodes = append(nodes, Node{
			Hostname: record[0],
			Port:     port,
			Username: record[2],
			Password: record[3],
		})
	}

	return nodes, nil
}

func checkNodes(nodes []Node, outputFile string) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10) // Limit to 10 concurrent goroutines

	for _, node := range nodes {
		wg.Add(1)
		sem <- struct{}{} // Acquire a token
		go func(n Node) {
			defer wg.Done()
			defer func() { <-sem }() // Release the token

			if isPortOpen(n.Hostname, n.Port) {
				if checkSSHAuth(n) {
					log.Printf("SSH Auth successful for (%s) on port (%s)", n.Hostname, n.Port)
					if outputFile != "" {
						appendToFile(outputFile, fmt.Sprintf("Success: %s (%s)\n", n.Port, n.Hostname))
					}
				} else {
					log.Printf("SSH Auth failed for (%s) on port %s", n.Hostname, n.Port)
					if outputFile != "" {
						appendToFile(outputFile, fmt.Sprintf("Failed: %s (%s)\n", n.Port, n.Hostname))
					}
				}
			} else {
				log.Printf("Port %s is not open for (%s)", n.Port, n.Hostname)
				if outputFile != "" {
					appendToFile(outputFile, fmt.Sprintf("Port Closed: %s (%s)\n", n.Port, n.Hostname))
				}
			}
		}(node)
	}

	wg.Wait()
}

func isPortOpen(ip, port string) bool {
	address := net.JoinHostPort(ip, port)
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}

func checkSSHAuth(node Node) bool {
	var password string

	if node.Password == "" {
		fmt.Print("Enter Password: ")
		passwordInput, err := readPassword()
		if err != nil {
			fmt.Println("Error reading password:", err)
			return false
		}
		password = passwordInput
	} else {
		password = node.Password
	}

	config := &ssh.ClientConfig{
		User: node.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	address := net.JoinHostPort(node.Hostname, node.Port)
	conn, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}

func readPassword() (string, error) {
	bytePassword, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", err
	}
	fmt.Println()
	return string(bytePassword), nil
}

func appendToFile(filename, text string) {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Error opening output file: %v", err)
	}
	defer f.Close()

	if _, err := f.WriteString(text); err != nil {
		log.Fatalf("Error writing to output file: %v", err)
	}
}
