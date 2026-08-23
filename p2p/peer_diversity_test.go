package p2p

import "testing"

func TestPeerNetworkGroupIPv4(t *testing.T) {
	group, err := PeerNetworkGroup("192.168.10.25:18700")
	if err != nil {
		t.Fatal(err)
	}
	if group != "ipv4:192.168" {
		t.Fatalf("unexpected IPv4 group: %s", group)
	}
}

func TestPeerNetworkGroupIPv6(t *testing.T) {
	group, err := PeerNetworkGroup("[2001:db8:1::10]:18700")
	if err != nil {
		t.Fatal(err)
	}
	if group != "ipv6:2001:0db8" {
		t.Fatalf("unexpected IPv6 group: %s", group)
	}
}

func TestPeerNetworkGroupHostnameNormalized(t *testing.T) {
	group, err := PeerNetworkGroup("Seed.Example.org:18700")
	if err != nil {
		t.Fatal(err)
	}
	if group != "dns:seed.example.org" {
		t.Fatalf("unexpected DNS group: %s", group)
	}
}

func TestCanAddPeerFromNetworkGroupAllowsDiversity(t *testing.T) {
	existing := []string{
		"10.1.1.1:18700",
		"10.1.2.1:18700",
		"10.2.1.1:18700",
	}

	if CanAddPeerFromNetworkGroup(existing, "10.1.9.9:18700") {
		t.Fatal("expected third peer from same IPv4 /16 group to be rejected")
	}
	if !CanAddPeerFromNetworkGroup(existing, "10.3.9.9:18700") {
		t.Fatal("expected peer from different IPv4 /16 group to be allowed")
	}
}

func TestCanAddPeerFromNetworkGroupRejectsInvalidAddress(t *testing.T) {
	if CanAddPeerFromNetworkGroup(nil, "not-an-address") {
		t.Fatal("invalid address should be rejected")
	}
}
