package stratum

import (
	"strings"
	"testing"
)

func TestParseWorkerIdentity(t *testing.T) {
	got, err := ParseWorkerIdentity("9ccdc094489874bed888ffe4bdf9b8298f4c5131.rig_01")
	if err != nil {
		t.Fatal(err)
	}
	if got.Wallet != "9ccdc094489874bed888ffe4bdf9b8298f4c5131" || got.Worker != "rig_01" {
		t.Fatalf("unexpected identity: %+v", got)
	}
}

func TestParseWorkerIdentityRejectsInvalidValues(t *testing.T) {
	wallet := "9ccdc094489874bed888ffe4bdf9b8298f4c5131"
	cases := map[string]string{
		"uppercase wallet": "9CCDC094489874bed888ffe4bdf9b8298f4c5131.rig_01",
		"non-hex wallet":   "gccdc094489874bed888ffe4bdf9b8298f4c5131.rig_01",
		"short wallet":     wallet[:39] + ".rig_01",
		"long wallet":      wallet + "0.rig_01",
		"empty worker":     wallet + ".",
		"long worker":      wallet + "." + strings.Repeat("a", 33),
		"additional dot":   wallet + ".rig.01",
		"leading space":    " " + wallet + ".rig_01",
		"trailing space":   wallet + ".rig_01 ",
		"control":          wallet + ".rig\n01",
		"bad worker char":  wallet + ".rig@01",
		"non ascii worker": wallet + ".rigé",
	}

	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseWorkerIdentity(input); err == nil {
				t.Fatalf("ParseWorkerIdentity(%q) unexpectedly succeeded", input)
			}
		})
	}
}
