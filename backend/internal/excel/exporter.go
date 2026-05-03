package excel

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/xuri/excelize/v2"
)

const summarySheetName = "Summary Chart"

// ExportMonthWorkbook creates an Excel workbook for one month plus chart data.
func ExportMonthWorkbook(month Month) ([]byte, error) {
	file := excelize.NewFile()
	defer file.Close()

	monthSheet := safeSheetName(month.ID)
	if err := file.SetSheetName("Sheet1", monthSheet); err != nil {
		return nil, err
	}

	styles, err := newExportStyles(file)
	if err != nil {
		return nil, err
	}

	if err := writeMonthSheet(file, monthSheet, month, styles); err != nil {
		return nil, err
	}
	if err := writeSummarySheet(file, summarySheetName, month, styles); err != nil {
		return nil, err
	}

	file.SetActiveSheet(0)
	buffer, err := file.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func ExportFileName(month Month) string {
	name := safeFileName(month.ID)
	if name == "" {
		name = "month"
	}
	return name + "-summary.xlsx"
}

type exportStyles struct {
	header    int
	number    int
	numberAlt int
	text      int
	textAlt   int
}

func newExportStyles(file *excelize.File) (exportStyles, error) {
	tableBorder := thinBorders("94A3B8")
	header, err := file.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"BE0037"}, Pattern: 1},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
			WrapText:   true,
		},
		Border: tableBorder,
	})
	if err != nil {
		return exportStyles{}, err
	}

	number, err := file.NewStyle(&excelize.Style{
		NumFmt: 3,
		Border: tableBorder,
		NegRed: true,
	})
	if err != nil {
		return exportStyles{}, err
	}

	numberAlt, err := file.NewStyle(&excelize.Style{
		NumFmt: 3,
		Fill:   excelize.Fill{Type: "pattern", Color: []string{"F8FAFC"}, Pattern: 1},
		Border: tableBorder,
		NegRed: true,
	})
	if err != nil {
		return exportStyles{}, err
	}

	text, err := file.NewStyle(&excelize.Style{
		Border: tableBorder,
	})
	if err != nil {
		return exportStyles{}, err
	}

	textAlt, err := file.NewStyle(&excelize.Style{
		Fill:   excelize.Fill{Type: "pattern", Color: []string{"F8FAFC"}, Pattern: 1},
		Border: tableBorder,
	})
	if err != nil {
		return exportStyles{}, err
	}

	return exportStyles{
		header:    header,
		number:    number,
		numberAlt: numberAlt,
		text:      text,
		textAlt:   textAlt,
	}, nil
}

func writeMonthSheet(file *excelize.File, sheet string, month Month, styles exportStyles) error {
	headers := monthHeaders(month.Dates)
	if err := setRow(file, sheet, 1, headers); err != nil {
		return err
	}

	lastColName, err := excelize.ColumnNumberToName(len(headers))
	if err != nil {
		return err
	}
	if err := file.SetCellStyle(sheet, "A1", lastColName+"1", styles.header); err != nil {
		return err
	}

	for rowIndex, member := range month.Members {
		row := memberExportRow(member, month.Dates)
		if err := setRow(file, sheet, rowIndex+2, row); err != nil {
			return err
		}
	}

	lastRow := len(month.Members) + 1
	noteColName, err := excelize.ColumnNumberToName(len(headers))
	if err != nil {
		return err
	}
	lastNumberColName, err := excelize.ColumnNumberToName(len(headers) - 1)
	if err != nil {
		return err
	}
	for rowIndex := 2; rowIndex <= lastRow; rowIndex++ {
		textStyle, numberStyle := rowStyles(rowIndex, styles)
		if err := file.SetCellStyle(sheet, fmt.Sprintf("A%d", rowIndex), fmt.Sprintf("A%d", rowIndex), textStyle); err != nil {
			return err
		}
		if err := file.SetCellStyle(sheet, fmt.Sprintf("B%d", rowIndex), fmt.Sprintf("%s%d", lastNumberColName, rowIndex), numberStyle); err != nil {
			return err
		}
		if err := file.SetCellStyle(sheet, fmt.Sprintf("%s%d", noteColName, rowIndex), fmt.Sprintf("%s%d", noteColName, rowIndex), textStyle); err != nil {
			return err
		}
	}

	_ = file.SetRowHeight(sheet, 1, 28)
	_ = file.SetColWidth(sheet, "A", "A", 24)
	_ = file.SetColWidth(sheet, "B", lastColName, 15)
	return file.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		XSplit:      1,
		YSplit:      1,
		TopLeftCell: "B2",
		ActivePane:  "bottomRight",
	})
}

func writeSummarySheet(file *excelize.File, sheet string, month Month, styles exportStyles) error {
	if _, err := file.NewSheet(sheet); err != nil {
		return err
	}

	rows := chartMembers(month.Members, 20)
	headers := []interface{}{"Name", "Fan Point Minus Debt", "Current Fans", "Debt", "Total Gain", "Avg Gain Per Week"}
	if err := setRow(file, sheet, 1, headers); err != nil {
		return err
	}
	if err := file.SetCellStyle(sheet, "A1", "F1", styles.header); err != nil {
		return err
	}

	for rowIndex, member := range rows {
		values := []interface{}{
			member.Name,
			fanPointMinusDebt(member),
			member.CurrentFans,
			member.Debt,
			member.TotalGain,
			member.AverageGain,
		}
		if err := setRow(file, sheet, rowIndex+2, values); err != nil {
			return err
		}
	}

	lastRow := len(rows) + 1
	for rowIndex := 2; rowIndex <= lastRow; rowIndex++ {
		textStyle, numberStyle := rowStyles(rowIndex, styles)
		if err := file.SetCellStyle(sheet, fmt.Sprintf("A%d", rowIndex), fmt.Sprintf("A%d", rowIndex), textStyle); err != nil {
			return err
		}
		if err := file.SetCellStyle(sheet, fmt.Sprintf("B%d", rowIndex), fmt.Sprintf("F%d", rowIndex), numberStyle); err != nil {
			return err
		}
	}

	_ = file.SetRowHeight(sheet, 1, 28)
	_ = file.SetColWidth(sheet, "A", "A", 24)
	_ = file.SetColWidth(sheet, "B", "F", 18)

	if len(rows) == 0 {
		return nil
	}

	quotedSheet := quoteSheetName(sheet)
	title := fmt.Sprintf("%s - Player Summary", month.Label)
	return file.AddChart(sheet, "G2", &excelize.Chart{
		Type: excelize.Bar,
		Series: []excelize.ChartSeries{
			{
				Name:       quotedSheet + "!$B$1",
				Categories: fmt.Sprintf("%s!$A$2:$A$%d", quotedSheet, lastRow),
				Values:     fmt.Sprintf("%s!$B$2:$B$%d", quotedSheet, lastRow),
			},
		},
		Title: []excelize.RichTextRun{{Text: title}},
		Legend: excelize.ChartLegend{
			Position:      "none",
			ShowLegendKey: false,
		},
		PlotArea: excelize.ChartPlotArea{
			ShowVal: false,
		},
		Dimension: excelize.ChartDimension{
			Width:  780,
			Height: 520,
		},
	})
}

func monthHeaders(dates []string) []interface{} {
	headers := []interface{}{"Name"}
	for index, date := range dates {
		headers = append(headers, date)
		if index < len(dates)-1 {
			headers = append(headers, fmt.Sprintf("แต้มแฟนสัปดาห์ที่ %d", index+1))
		}
	}
	headers = append(headers, "แต้มสะสมรวม 2 สัปดาห์", "แต้มสะสมรวม 4 สัปดาห์", "Debt", "หมายเหตุ")
	return headers
}

func memberExportRow(member Member, dates []string) []interface{} {
	row := []interface{}{member.Name}
	snapshots := snapshotByDate(member.Snapshots)
	gains := gainByDate(member.WeeklyGains)

	for index, date := range dates {
		row = append(row, snapshots[date])
		if index < len(dates)-1 {
			row = append(row, gains[dates[index+1]])
		}
	}

	row = append(row, sumGains(member.WeeklyGains, 2), sumGains(member.WeeklyGains, 4), member.Debt, member.Note)
	return row
}

func snapshotByDate(snapshots []Snapshot) map[string]int64 {
	values := make(map[string]int64, len(snapshots))
	for _, snapshot := range snapshots {
		values[snapshot.Date] = snapshot.Fans
	}
	return values
}

func gainByDate(gains []WeekGain) map[string]int64 {
	values := make(map[string]int64, len(gains))
	for _, gain := range gains {
		values[gain.Date] = gain.Gain
	}
	return values
}

func sumGains(gains []WeekGain, limit int) int64 {
	var total int64
	for index, gain := range gains {
		if index >= limit {
			break
		}
		total += gain.Gain
	}
	return total
}

func chartMembers(members []Member, limit int) []Member {
	rows := append([]Member(nil), members...)
	sort.SliceStable(rows, func(i, j int) bool {
		return fanPointMinusDebt(rows[i]) > fanPointMinusDebt(rows[j])
	})
	if len(rows) > limit {
		return rows[:limit]
	}
	return rows
}

func fanPointMinusDebt(member Member) int64 {
	return member.CurrentFans + member.Debt
}

func rowStyles(rowIndex int, styles exportStyles) (int, int) {
	if rowIndex%2 == 0 {
		return styles.text, styles.number
	}
	return styles.textAlt, styles.numberAlt
}

func setRow(file *excelize.File, sheet string, rowIndex int, values []interface{}) error {
	cell, err := excelize.CoordinatesToCellName(1, rowIndex)
	if err != nil {
		return err
	}
	return file.SetSheetRow(sheet, cell, &values)
}

func thinBorders(color string) []excelize.Border {
	return []excelize.Border{
		{Type: "left", Color: color, Style: 1},
		{Type: "right", Color: color, Style: 1},
		{Type: "top", Color: color, Style: 1},
		{Type: "bottom", Color: color, Style: 1},
	}
}

func safeSheetName(value string) string {
	name := strings.TrimSpace(value)
	if name == "" {
		name = "Month"
	}
	replacer := strings.NewReplacer("[", "(", "]", ")", ":", "-", "*", "-", "?", "", "/", "-", "\\", "-")
	name = replacer.Replace(name)
	if len([]rune(name)) > 31 {
		name = string([]rune(name)[:31])
	}
	return name
}

func safeFileName(value string) string {
	name := strings.TrimSpace(value)
	name = regexp.MustCompile(`[^a-zA-Z0-9._-]+`).ReplaceAllString(name, "-")
	return strings.Trim(name, "-.")
}

func quoteSheetName(name string) string {
	return "'" + strings.ReplaceAll(name, "'", "''") + "'"
}
