package enrollment

import "testing"

func TestContractVersions(t *testing.T) {
	if ProtocolVersion != "1.0.0" || AdmissionContractVersion != "1.0.0" {
		t.Fatal("unexpected contract versions")
	}
}
