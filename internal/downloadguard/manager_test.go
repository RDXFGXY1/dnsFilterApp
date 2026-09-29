package downloadguard

import "testing"

func TestRiskSignalsAreExplainable(t *testing.T) {
	score, reasons := scoreFile("invoice.pdf.exe", ".exe", "windows-executable", "unsigned")
	if score < 50 {
		t.Fatalf("expected high risk, got %d", score)
	}
	if len(reasons) < 2 {
		t.Fatalf("expected multiple reasons: %+v", reasons)
	}
}
func TestValidSignatureReducesRisk(t *testing.T) {
	unsigned, _ := scoreFile("setup.exe", ".exe", "windows-executable", "unsigned")
	signed, _ := scoreFile("setup.exe", ".exe", "windows-executable", "valid")
	if signed >= unsigned {
		t.Fatalf("valid signature did not reduce risk: signed=%d unsigned=%d", signed, unsigned)
	}
}
func TestHeaderTypeDetection(t *testing.T) {
	if got := detectType([]byte{'M', 'Z', 0, 0}, ".pdf"); got != "windows-executable" {
		t.Fatalf("got %s", got)
	}
	if got := detectType([]byte("%PDF-1.7"), ".pdf"); got != "pdf-document" {
		t.Fatalf("got %s", got)
	}
}
