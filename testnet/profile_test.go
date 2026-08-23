package testnet

import "testing"

func TestDefaultProfile(t *testing.T) {
	p := DefaultProfile()
	if p.Slug != Slug || p.P2PPort == p.RPCPort || p.DataDir != DefaultDataDir {
		t.Fatalf("unexpected testnet defaults: %+v", p)
	}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestProfileSeedValidation(t *testing.T) {
	p := DefaultProfile()
	p.Seeds = []string{"seed1.example.org:28444", "seed2.example.org:28444"}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	p.Seeds = append(p.Seeds, p.Seeds[0])
	if err := p.Validate(); err == nil {
		t.Fatal("expected duplicate seed rejection")
	}
}

func TestProfileRejectsPortCollision(t *testing.T) {
	p := DefaultProfile()
	p.RPCPort = p.P2PPort
	if err := p.Validate(); err == nil {
		t.Fatal("expected port collision rejection")
	}
}

func TestPublicLaunchRequiresTwoPublicSeeds(t *testing.T) {
	p := DefaultProfile()
	p.Seeds = []string{"seed1.sudharma.net:28444"}
	if err := p.ValidatePublicLaunch(); err == nil {
		t.Fatal("expected public launch to require two seed nodes")
	}
	p.Seeds = append(p.Seeds, "seed2.sudharma.net:28444")
	if err := p.ValidatePublicLaunch(); err != nil {
		t.Fatal(err)
	}
}

func TestPublicLaunchRejectsClearlyPrivateOrReservedSeeds(t *testing.T) {
	cases := []string{
		"127.0.0.1:28444",
		"10.0.0.10:28444",
		"localhost:28444",
		"seed.local:28444",
		"seed.example:28444",
	}
	for _, seed := range cases {
		p := DefaultProfile()
		p.Seeds = []string{seed, "seed2.sudharma.net:28444"}
		if err := p.ValidatePublicLaunch(); err == nil {
			t.Fatalf("expected %q to be rejected as a public seed", seed)
		}
	}
}
