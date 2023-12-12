package util

import (
	"log"
	"os/user"
)

// Check if the current user is root or not
// fail if user is not root
func CheckUser() {
	user, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	if user.Username == "root" {
		return
	}
	log.Fatal("exiting, script needs to be run as root, current user is ", user.Username)
}
