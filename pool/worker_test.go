package pool

import "testing"

func TestParseWorkerIdentity(t *testing.T) {
	identity, err := ParseWorkerIdentity("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.rig1")
	if err != nil {
		t.Fatal(err)
	}
	if identity.Address != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("address = %q", identity.Address)
	}
	if identity.WorkerName != "rig1" {
		t.Fatalf("worker = %q", identity.WorkerName)
	}

	defaultWorker, err := ParseWorkerIdentity("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	if err != nil {
		t.Fatal(err)
	}
	if defaultWorker.WorkerName != "default" {
		t.Fatalf("worker = %q", defaultWorker.WorkerName)
	}
}

func TestParseWorkerIdentityRejectsInvalidLogin(t *testing.T) {
	if _, err := ParseWorkerIdentity("not-a-wallet"); err == nil {
		t.Fatal("expected invalid login error")
	}
	if _, err := ParseWorkerIdentity("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa."); err == nil {
		t.Fatal("expected empty worker error")
	}
}
