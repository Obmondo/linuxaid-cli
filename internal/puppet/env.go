package puppet

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// LoadPuppetEnv doesnt throw error if the file doesnt exist
func LoadPuppetEnv() error {
	err := godotenv.Load("/etc/default/run_puppet")
	if err == nil || os.IsNotExist(err) {
		return nil
	}

	return fmt.Errorf("could not load /etc/default/run_puppet: %w", err)
}
