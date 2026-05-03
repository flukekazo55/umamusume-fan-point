package excel

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

func TestExportMonthWorkbookIncludesMonthSheetAndChart(t *testing.T) {
	month := NormalizeMonth(Month{
		ID:        "May2026",
		Label:     "May 2026",
		StartDate: "2026-05-01",
		EndDate:   "2026-05-31",
		Dates:     []string{"2026-05-01", "2026-05-10", "2026-05-17"},
		Members: []Member{
			{
				Name: "PlayerA",
				Debt: -100,
				Snapshots: []Snapshot{
					{Date: "2026-05-01", Fans: 1000},
					{Date: "2026-05-10", Fans: 1500},
					{Date: "2026-05-17", Fans: 2100},
				},
			},
			{
				Name: "PlayerB",
				Snapshots: []Snapshot{
					{Date: "2026-05-01", Fans: 900},
					{Date: "2026-05-10", Fans: 1200},
					{Date: "2026-05-17", Fans: 1800},
				},
			},
		},
	})

	data, err := ExportMonthWorkbook(month)
	if err != nil {
		t.Fatalf("ExportMonthWorkbook() error = %v", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("open exported workbook: %v", err)
	}

	var workbookXML string
	var sharedStringsXML string
	var chartXML string
	var hasChart bool
	for _, file := range reader.File {
		if file.Name == "xl/workbook.xml" {
			workbookXML = readZipFile(t, file)
		}
		if file.Name == "xl/sharedStrings.xml" {
			sharedStringsXML = readZipFile(t, file)
		}
		if strings.HasPrefix(file.Name, "xl/charts/chart") {
			hasChart = true
			chartXML = readZipFile(t, file)
		}
	}

	if !strings.Contains(workbookXML, `name="May2026"`) {
		t.Fatalf("exported workbook does not include selected month sheet: %s", workbookXML)
	}
	if !strings.Contains(workbookXML, `name="Summary Chart"`) {
		t.Fatalf("exported workbook does not include summary sheet: %s", workbookXML)
	}
	if !hasChart {
		t.Fatal("exported workbook does not include an Excel chart")
	}
	if !strings.Contains(sharedStringsXML, "Fan Point Minus Debt") {
		t.Fatalf("summary sheet does not include adjusted fan point header: %s", sharedStringsXML)
	}
	if !strings.Contains(chartXML, "$B$2:$B$") {
		t.Fatalf("summary chart is not using fan point minus debt values: %s", chartXML)
	}
}

func readZipFile(t *testing.T, file *zip.File) string {
	t.Helper()

	handle, err := file.Open()
	if err != nil {
		t.Fatalf("open %s: %v", file.Name, err)
	}
	defer handle.Close()

	buffer := new(bytes.Buffer)
	if _, err := buffer.ReadFrom(handle); err != nil {
		t.Fatalf("read %s: %v", file.Name, err)
	}
	return buffer.String()
}
