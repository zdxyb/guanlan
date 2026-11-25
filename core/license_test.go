package core

import (
	"strings"
	"testing"
	"time"
)

func TestLicenseRoundTrip(t *testing.T) {
	code := issueLicense(0, machineKey())
	exp, err := checkLicenseCode(strings.ToLower(code))
	if err != nil {
		t.Fatal(err)
	}
	if exp != 0 {
		t.Fatalf("exp = %d", exp)
	}
	if _, err := checkLicenseCode("GL-AAAA-BBBB-CCCC"); err == nil {
		t.Fatal("expected invalid code")
	}
	other := issueLicense(0, machineKey())
	if other == code {
		t.Fatal("expected a different code")
	}
	if _, err := checkLicenseCode(other); err != nil {
		t.Fatal(err)
	}
}

func TestLicenseOtherMachine(t *testing.T) {
	other := make([]byte, 8)
	copy(other, machineKey())
	other[0] ^= 0xff
	code := issueLicense(0, other)
	_, err := checkLicenseCode(code)
	if err != errLicenseMachine {
		t.Fatalf("err = %v", err)
	}
}

func TestLicenseExpired(t *testing.T) {
	code := issueLicense(time.Now().Add(-time.Hour).Unix(), machineKey())
	_, err := checkLicenseCode(code)
	if err != errLicenseExpired {
		t.Fatalf("err = %v", err)
	}
}
