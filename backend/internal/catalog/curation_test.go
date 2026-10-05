package catalog

import "testing"

func TestCurationManifestDeniesPermissions(t *testing.T) {
	for _, access := range []map[string]bool{nil, {"public": true}, {"public": false}, {"read": false, "playback": false, "download": false, "acquisition": false, "relay": false, "schedule": false, "public": false, "extra": false}} {
		if noCurationAccess(access) {
			t.Fatal("invalid access grant accepted", access)
		}
	}
}
