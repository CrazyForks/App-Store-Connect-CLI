package cmd

import "testing"

func TestNormalizeFinanceReportType(t *testing.T) {
	reportType, err := normalizeFinanceReportType("financial")
	if err != nil {
		t.Fatalf("expected report type to parse, got %v", err)
	}
	if string(reportType) != "FINANCIAL" {
		t.Fatalf("expected FINANCIAL, got %q", reportType)
	}

	_, err = normalizeFinanceReportType("invalid")
	if err == nil {
		t.Fatal("expected error for invalid report type")
	}
}

func TestNormalizeFinanceReportDate(t *testing.T) {
	date, err := normalizeFinanceReportDate("2025-12")
	if err != nil {
		t.Fatalf("expected date to parse, got %v", err)
	}
	if date != "2025-12" {
		t.Fatalf("expected date to be 2025-12, got %q", date)
	}

	_, err = normalizeFinanceReportDate("2025-13")
	if err == nil {
		t.Fatal("expected error for invalid date")
	}
}
