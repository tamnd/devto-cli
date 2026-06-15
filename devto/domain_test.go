package devto

import (
	"testing"

	"github.com/tamnd/any-cli/kit"
)

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "devto" {
		t.Errorf("Scheme = %q, want devto", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "devto" {
		t.Errorf("Identity.Binary = %q, want devto", info.Identity.Binary)
	}
}

func TestHostWiring(t *testing.T) {
	h, err := kit.Open()
	if err != nil {
		t.Fatal(err)
	}
	domains := h.Domains()
	found := false
	for _, d := range domains {
		if d == "devto" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("devto domain not mounted; got %v", domains)
	}
	info, ok := h.Domain("devto")
	if !ok {
		t.Fatal("h.Domain(devto) returned false")
	}
	if info.Identity.Binary != "devto" {
		t.Errorf("Binary = %q, want devto", info.Identity.Binary)
	}
}
