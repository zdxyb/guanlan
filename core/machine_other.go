//go:build !windows

package core

import "os"

func machineFingerprint() string {
	host, _ := os.Hostname()
	return host
}
