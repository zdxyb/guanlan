package core

import (
	"os"

	"golang.org/x/sys/windows/registry"
)

func machineFingerprint() string {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Cryptography`, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		host, _ := os.Hostname()
		return host
	}
	defer key.Close()
	guid, _, err := key.GetStringValue("MachineGuid")
	if err != nil || guid == "" {
		host, _ := os.Hostname()
		return host
	}
	return guid
}
