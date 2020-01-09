package suffixdomain

import "testing"

func TestGenerateDomain(t *testing.T) {
	domain, err := GenerateDomain("192.168.2.203")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(domain)
}
