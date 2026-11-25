package core

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	errInvalidLicense = errors.New("授权码不正确")
	errLicenseExpired = errors.New("授权码已过期")
	errLicenseMachine = errors.New("授权码与这台电脑不匹配")
	errBadMachine     = errors.New("电脑识别码不正确")
	licenseSecret     = []byte("guanlan-downloader-license-v1-7c3e9a1b")
	licenseEncoding   = base32.StdEncoding.WithPadding(base32.NoPadding)

	licenseMu  sync.RWMutex
	licenseOK  bool
	licenseExp int64
)

func issueLicense(exp int64, machine []byte) string {
	payload := make([]byte, 17)
	payload[0] = 3
	binary.BigEndian.PutUint32(payload[1:], uint32(exp))
	copy(payload[5:13], machine)
	if _, err := rand.Read(payload[13:]); err != nil {
		binary.BigEndian.PutUint32(payload[13:], uint32(time.Now().UnixNano()))
	}
	raw := append(payload, licenseMAC(payload)...)
	return formatLicense(licenseEncoding.EncodeToString(raw))
}

// IssueLicense returns a code that only works on the computer identified by machineCode.
// days <= 0 means the code does not expire.
func IssueLicense(days int, machineCode string) (string, error) {
	machine, err := parseMachineCode(machineCode)
	if err != nil {
		return "", err
	}
	var exp int64
	if days > 0 {
		exp = time.Now().Add(time.Duration(days) * 24 * time.Hour).Unix()
	}
	return issueLicense(exp, machine), nil
}

func machineKey() []byte {
	sum := sha256.Sum256([]byte("guanlan-machine:" + machineFingerprint()))
	return sum[:8]
}

func MachineCode() string {
	return formatMachine(licenseEncoding.EncodeToString(machineKey()))
}

func formatMachine(raw string) string {
	var b strings.Builder
	b.WriteString("MC")
	for i := 0; i < len(raw); i++ {
		if i%4 == 0 {
			b.WriteByte('-')
		}
		b.WriteByte(raw[i])
	}
	return b.String()
}

func parseMachineCode(code string) ([]byte, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	code = strings.ReplaceAll(code, "-", "")
	code = strings.ReplaceAll(code, " ", "")
	code = strings.TrimPrefix(code, "MC")
	raw, err := licenseEncoding.DecodeString(code)
	if err != nil || len(raw) != 8 {
		return nil, errBadMachine
	}
	return raw, nil
}

func licenseMAC(payload []byte) []byte {
	mac := hmac.New(sha256.New, licenseSecret)
	mac.Write(payload)
	return mac.Sum(nil)[:8]
}

func formatLicense(raw string) string {
	var b strings.Builder
	b.WriteString("GL")
	for i := 0; i < len(raw); i++ {
		if i%4 == 0 {
			b.WriteByte('-')
		}
		b.WriteByte(raw[i])
	}
	return b.String()
}

func normalizeLicense(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	code = strings.ReplaceAll(code, "-", "")
	code = strings.ReplaceAll(code, " ", "")
	code = strings.TrimPrefix(code, "GL")
	return code
}

func checkLicenseCode(code string) (int64, error) {
	raw, err := licenseEncoding.DecodeString(normalizeLicense(code))
	if err != nil {
		return 0, errInvalidLicense
	}
	if len(raw) != 25 || raw[0] != 3 {
		return 0, errInvalidLicense
	}
	payload := raw[:17]
	if !hmac.Equal(raw[len(payload):], licenseMAC(payload)) {
		return 0, errInvalidLicense
	}
	exp := int64(binary.BigEndian.Uint32(payload[1:5]))
	if !bytes.Equal(payload[5:13], machineKey()) {
		return exp, errLicenseMachine
	}
	if exp > 0 && time.Now().Unix() > exp {
		return exp, errLicenseExpired
	}
	return exp, nil
}

func licenseFile() string {
	return filepath.Join(appOnce.UserDir, "license.json")
}

func initLicense() {
	data, err := os.ReadFile(licenseFile())
	if err != nil {
		return
	}
	var stored struct {
		Code string `json:"code"`
	}
	if json.Unmarshal(data, &stored) != nil {
		return
	}
	exp, err := checkLicenseCode(stored.Code)
	if err != nil {
		return
	}
	licenseMu.Lock()
	licenseOK = true
	licenseExp = exp
	licenseMu.Unlock()
}

func Licensed() bool {
	licenseMu.RLock()
	ok := licenseOK
	exp := licenseExp
	licenseMu.RUnlock()
	if !ok {
		return false
	}
	if exp > 0 && time.Now().Unix() > exp {
		return false
	}
	return true
}

func LicenseExpires() int64 {
	licenseMu.RLock()
	defer licenseMu.RUnlock()
	if !licenseOK {
		return 0
	}
	return licenseExp
}

func activateLicense(code string) (int64, error) {
	exp, err := checkLicenseCode(code)
	if err != nil {
		return 0, err
	}
	stored, err := json.Marshal(map[string]string{"code": formatLicense(normalizeLicense(code))})
	if err != nil {
		return 0, err
	}
	if err := os.WriteFile(licenseFile(), stored, 0644); err != nil {
		return 0, err
	}
	licenseMu.Lock()
	licenseOK = true
	licenseExp = exp
	licenseMu.Unlock()
	return exp, nil
}

func formatLicenseExpiry(exp int64) string {
	if exp <= 0 {
		return ""
	}
	return time.Unix(exp, 0).Format("2006-01-02")
}
