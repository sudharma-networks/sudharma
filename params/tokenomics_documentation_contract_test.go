package params

import (
	"os"
	"strings"
	"testing"
)

func TestTokenomicsPublicWordingDistinguishesTestnetAndMainnetCaps(t *testing.T) {
	if MaxSupplySUDH != 51_000_000_000 {
		t.Fatalf("public-testnet supply constant changed: got %d", MaxSupplySUDH)
	}
	if MainnetMaxSupplySUDH != 51_000_000 {
		t.Fatalf("mainnet candidate supply constant changed: got %d", MainnetMaxSupplySUDH)
	}

	readme, err := os.ReadFile("../README.md")
	if err != nil {
		t.Fatal(err)
	}
	readmeText := string(readme)
	if strings.Contains(readmeText, "| Maximum Supply (Hard Cap) | 51,000,000,000 SUDH |") {
		t.Fatal("README still presents the 51B public-testnet cap as a generic network hard cap")
	}
	for _, required := range []string{
		"| Public testnet maximum supply (hard cap) | 51,000,000,000 SUDH |",
		"| Mainnet candidate maximum supply (hard cap) | 51,000,000 SUDH |",
	} {
		if !strings.Contains(readmeText, required) {
			t.Fatalf("README missing explicit tokenomics wording %q", required)
		}
	}

	project, err := os.ReadFile("../web/lib/project.ts")
	if err != nil {
		t.Fatal(err)
	}
	projectText := string(project)
	if strings.Contains(projectText, `["Maximum supply (hard cap)", "51,000,000,000 SUDH"]`) {
		t.Fatal("website still presents the 51B public-testnet cap as a generic network hard cap")
	}
	for _, required := range []string{
		`["Public testnet maximum supply (hard cap)", "51,000,000,000 SUDH"]`,
		`["Mainnet candidate maximum supply (hard cap)", "51,000,000 SUDH"]`,
	} {
		if !strings.Contains(projectText, required) {
			t.Fatalf("website project parameters missing explicit tokenomics wording %q", required)
		}
	}
}
