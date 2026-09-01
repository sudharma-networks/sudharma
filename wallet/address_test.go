package wallet

import "testing"

func TestValidateAddressAcceptsCanonicalForm(t *testing.T) {
	address := "16d7dc9ec0495109007860a584c7cf9055da9abf"
	if err := ValidateAddress(address); err != nil {
		t.Fatalf("valid address rejected: %v", err)
	}
}

func TestValidateAddressRejectsUppercase(t *testing.T) {
	if err := ValidateAddress("16D7DC9EC0495109007860A584C7CF9055DA9ABF"); err == nil {
		t.Fatal("uppercase address was accepted")
	}
}

func TestValidateAddressRejectsOversizedReceiver(t *testing.T) {
	oversized := stringsRepeat("a", AddressLength+1)
	if err := ValidateAddress(oversized); err == nil {
		t.Fatal("oversized address was accepted")
	}
}

func stringsRepeat(value string, count int) string {
	result := make([]byte, 0, len(value)*count)
	for i := 0; i < count; i++ {
		result = append(result, value...)
	}
	return string(result)
}
