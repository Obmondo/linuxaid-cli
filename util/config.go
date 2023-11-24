package util

import (
	"log"
	"os"
	"strings"
)

var (
	host = func() string {
		host, err := os.Hostname()
		if err != nil {
			log.Println("Error getting hostname: " + err.Error() + "\n")
			os.Exit(1)
		}
		return strings.ToLower(host)
	}()

	customer = func() string {
		var customer string
		if customer == "" {
			return "__CUSTOMER_ID__"
		}
		return customer
	}()
)

// GetHost returns the host value
func GetHost() string {
	return host
}

// GetCustomer returns the customer value
func GetCustomer() string {
	return customer
}
