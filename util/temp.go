package util

import (
	"log"
	"os"
)

func TempDir() string {
	dir, err := os.MkdirTemp("", "")
	if err != nil {
		log.Fatal(err)
	}

	return dir
}
