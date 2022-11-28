package utils

import (
	"os"
)

// MyUserAndGroup returns the UID and GID of this process.
func MyUserAndGroup() (int, int) {
	return os.Getuid(), os.Getgid()
}
