package testnet

import "testing"

func TestLaunchManifestRequiresPublicReadyProfile(t *testing.T) {
	profile := DefaultProfile()
	if _, err := NewLaunchManifest(profile); err == nil {
		t.Fatal("expected launch manifest to reject profile without public seeds")
	}

	profile.Seeds = []string{"seed1.example.org:28444", "seed2.example.org:28444"}
	manifest, err := NewLaunchManifest(profile)
	if err != nil { t.Fatal(err) }
	if manifest.ProtocolNetworkID != ProtocolNetworkID || manifest.GenesisHash != GenesisHash() {
		t.Fatalf("manifest identity mismatch: %+v", manifest)
	}
	if len(manifest.Seeds) != MinimumSeedNodes {
		t.Fatalf("unexpected seed count: %d", len(manifest.Seeds))
	}
}
