package params

import (
	"os"
	"strings"
	"testing"
)

func TestTokenomicsPublicWordingLocksFinalMainnetPolicy(t *testing.T) {
	if MaxSupplySUDH != 51_000_000_000 {
		t.Fatalf("public-testnet legacy supply constant changed: got %d", MaxSupplySUDH)
	}
	if MainnetMaxSupplySUDH != 51_000_000 {
		t.Fatalf("final mainnet supply constant changed: got %d", MainnetMaxSupplySUDH)
	}
	if MainnetEpochCount != 40 {
		t.Fatalf("final mainnet epoch count changed: got %d", MainnetEpochCount)
	}
	if MainnetEpochLength != 131_490 {
		t.Fatalf("final mainnet epoch length changed: got %d", MainnetEpochLength)
	}
	if MainnetFinalSubsidyHeight != 5_259_600 {
		t.Fatalf("final mainnet subsidy height changed: got %d", MainnetFinalSubsidyHeight)
	}

	var total uint64
	for _, epoch := range MainnetEmissionEpochs {
		total += epoch.Issuance
	}
	if total != MainnetMaxSupply {
		t.Fatalf("final mainnet emission table must total exactly 51M SUDH: got %d base units", total)
	}

	readme, err := os.ReadFile("../README.md")
	if err != nil {
		t.Fatal(err)
	}
	readmeText := string(readme)
	for _, forbidden := range []string{
		"| Maximum Supply (Hard Cap) | 51,000,000,000 SUDH |",
		"Mainnet candidate maximum supply",
		"mainnet candidate policy",
	} {
		if strings.Contains(readmeText, forbidden) {
			t.Fatalf("README still contains stale or ambiguous tokenomics wording %q", forbidden)
		}
	}
	for _, required := range []string{
		"| Public testnet maximum supply (legacy hard cap) | 51,000,000,000 SUDH |",
		"| Mainnet maximum supply (final monetary policy) | 51,000,000 SUDH |",
		"| Mainnet subsidy-bearing blocks | 5,259,600 |",
		"| Mainnet emission epochs | 40 quarterly epochs |",
		"| Mainnet nominal subsidy period | ~10 target years |",
		"| Mainnet subsidy after height 5,259,600 | 0 SUDH |",
	} {
		if !strings.Contains(readmeText, required) {
			t.Fatalf("README missing final mainnet tokenomics wording %q", required)
		}
	}

	project, err := os.ReadFile("../web/lib/project.ts")
	if err != nil {
		t.Fatal(err)
	}
	projectText := string(project)
	for _, forbidden := range []string{
		`["Maximum supply (hard cap)", "51,000,000,000 SUDH"]`,
		"Mainnet candidate maximum supply",
	} {
		if strings.Contains(projectText, forbidden) {
			t.Fatalf("website still contains stale or ambiguous tokenomics wording %q", forbidden)
		}
	}
	for _, required := range []string{
		`["Public testnet maximum supply (legacy hard cap)", "51,000,000,000 SUDH"]`,
		`["Mainnet maximum supply (final monetary policy)", "51,000,000 SUDH"]`,
		`["Mainnet subsidy-bearing blocks", "5,259,600"]`,
		`["Mainnet emission epochs", "40 quarterly epochs"]`,
		`["Mainnet nominal subsidy period", "~10 target years"]`,
		`["Mainnet subsidy after height 5,259,600", "0 SUDH"]`,
	} {
		if !strings.Contains(projectText, required) {
			t.Fatalf("website project parameters missing final mainnet tokenomics wording %q", required)
		}
	}

	sourceOfTruth, err := os.ReadFile("../docs/audits/2026-09-01-tokenomics-source-of-truth-kk.md")
	if err != nil {
		t.Fatal(err)
	}
	sourceText := string(sourceOfTruth)
	for _, forbidden := range []string{
		"mainnet candidate monetary policy",
		"Mainnet candidate",
		"mainnet candidate cap",
	} {
		if strings.Contains(sourceText, forbidden) {
			t.Fatalf("tokenomics source of truth still contains stale candidate wording %q", forbidden)
		}
	}
	for _, required := range []string{
		"**51,000,000** (51M)",
		"**5,259,600 subsidy-bearing blocks**",
		"**40 quarterly epochs**",
		"**10 target years**",
		"subsidy is permanently **0** after height 5,259,600",
	} {
		if !strings.Contains(sourceText, required) {
			t.Fatalf("tokenomics source of truth missing final-policy statement %q", required)
		}
	}
}
