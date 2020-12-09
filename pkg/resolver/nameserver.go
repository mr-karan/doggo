package resolver

import (
	"bufio"
	"os"
	"strings"
)

// GetDefaultNameserver reads `/etc/resolv.conf` to determine the default
// nameserver configured. Returns an error if unable to parse or
// no nameserver specified. It returns as soon as it finds a line
// with `nameserver` prefix.
// An example format:
// `nameserver 127.0.0.1`

func GetDefaultNameserver() (string, error) {
	file, err := os.Open("/etc/resolv.conf")
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "nameserver") {
			return strings.Fields(scanner.Text())[1], nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", err
}
