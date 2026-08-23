package testnet

import "testing"

func TestDefaultProfile(t *testing.T) {
	p := DefaultProfile()
	if p.Slug != Slug || p.P2PPort == p.RPCPort || p.DataDir != DefaultDataDir {
		t.Fatalf("unexpected testnet defaults: %+v", p)
	}
	if err := p.Validate(); err != nil { t.Fatal(err) }
}

func TestProfileSeedValidation(t *testing.T) {
	p := DefaultProfile()
	p.Seeds = []string{"seed1.example.org:28444", "seed2.example.org:28444"}
	if err := p.Validate(); err != nil { t.Fatal(err) }
	p.Seeds = append(p.Seeds, p.Seeds[0])
	if err := p.Validate(); err == nil { t.Fatal("expected duplicate seed rejection") }
}

func TestProfileRejectsPortCollision(t *testing.T) {
	p := DefaultProfile()
	p.RPCPort = p.P2PPort
	if err := p.Validate(); err == nil { t.Fatal("expected port collision rejection") }
}
