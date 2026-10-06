package community

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRecoveryJournalIndependentAndLatestRestriction(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "private")
	now := time.Now().UTC()
	id := Hash("user")
	checkpoint := Checkpoint{Version: 1, Deployment: "brand", At: now.Add(-time.Hour), Restrictions: []Restriction{{User: id, Until: now.Add(24 * time.Hour)}}}
	for _, event := range []RecoveryEvent{
		{Version: 1, Deployment: "brand", At: now.Add(-time.Minute), Kind: "account", ID: id},
		{Version: 1, Deployment: "brand", At: now, Kind: "restriction", ID: id, Until: now.Add(time.Hour)},
	} {
		if err := WriteRecoveryEvent(directory, event); err != nil {
			t.Fatal(err)
		}
	}
	result, err := MergeRecoveryJournal(checkpoint, directory, now)
	if err != nil || len(result.Deletions) != 1 || len(result.Restrictions) != 1 || !result.Restrictions[0].Until.Equal(now.Add(time.Hour)) {
		t.Fatalf("current journal lost: %+v %v", result, err)
	}
	entries, _ := os.ReadDir(directory)
	path := filepath.Join(directory, entries[0].Name())
	if err = os.WriteFile(path, []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = MergeRecoveryJournal(checkpoint, directory, now); err == nil {
		t.Fatal("corrupt journal accepted")
	}
	public := filepath.Join(t.TempDir(), "public")
	if err = os.Mkdir(public, 0755); err != nil {
		t.Fatal(err)
	}
	if err = WriteRecoveryEvent(public, RecoveryEvent{Version: 1, Deployment: "brand", At: now, Kind: "account", ID: id}); err == nil {
		t.Fatal("public journal accepted")
	}
}
