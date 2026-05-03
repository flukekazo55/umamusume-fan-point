package excel

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Parser struct {
	path string
}

func NewParser(path string) *Parser {
	return &Parser{path: path}
}

func (p *Parser) Load() (*Workbook, error) {
	reader, err := zip.OpenReader(p.path)
	if err != nil {
		return nil, fmt.Errorf("open workbook: %w", err)
	}
	defer reader.Close()

	files := indexFiles(reader.File)

	shared, err := readSharedStrings(files)
	if err != nil {
		return nil, err
	}

	sheets, err := readWorkbookSheets(files)
	if err != nil {
		return nil, err
	}

	months := make([]Month, 0, len(sheets))
	oldMembers := make([]OldMember, 0)
	seenMembers := map[string]struct{}{}
	var totalCurrent int64

	for _, sheet := range sheets {
		cells, err := readSheet(files, sheet.Path, shared)
		if err != nil {
			return nil, err
		}

		if strings.EqualFold(sheet.Name, "Old member") {
			oldMembers = parseOldMembers(cells)
			continue
		}
		if strings.Contains(sheet.Name, "เก่า") {
			continue
		}

		month, ok := parseMonth(sheet.Name, cells)
		if !ok {
			continue
		}
		months = append(months, month)
		for _, member := range month.Members {
			seenMembers[member.Name] = struct{}{}
		}
		if len(months) == 1 {
			totalCurrent = month.Stats.TotalCurrent
		}
	}

	latestID := ""
	if len(months) > 0 {
		latestID = months[0].ID
	}

	return &Workbook{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		LatestID:    latestID,
		Months:      months,
		OldMembers:  oldMembers,
		Summary: BookSummary{
			MonthCount:       len(months),
			TrackedMembers:   len(seenMembers),
			TotalCurrentFans: totalCurrent,
		},
	}, nil
}

type sheetInfo struct {
	Name string
	Path string
}

type workbookXML struct {
	Sheets []workbookSheetXML `xml:"sheets>sheet"`
}

type workbookSheetXML struct {
	Name string `xml:"name,attr"`
	RID  string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
}

type relsXML struct {
	Relationships []relationshipXML `xml:"Relationship"`
}

type relationshipXML struct {
	ID     string `xml:"Id,attr"`
	Target string `xml:"Target,attr"`
}

type sheetXML struct {
	Rows []rowXML `xml:"sheetData>row"`
}

type rowXML struct {
	R     int       `xml:"r,attr"`
	Cells []cellXML `xml:"c"`
}

type cellXML struct {
	Ref       string       `xml:"r,attr"`
	Type      string       `xml:"t,attr"`
	Value     string       `xml:"v"`
	InlineStr inlineString `xml:"is"`
}

type inlineString struct {
	Text string `xml:"t"`
}

type cellValue struct {
	Raw string
	Col int
}

func indexFiles(files []*zip.File) map[string]*zip.File {
	index := make(map[string]*zip.File, len(files))
	for _, file := range files {
		index[file.Name] = file
	}
	return index
}

func readSharedStrings(files map[string]*zip.File) ([]string, error) {
	file, ok := files["xl/sharedStrings.xml"]
	if !ok {
		return nil, nil
	}

	var data struct {
		Items []struct {
			Texts []string `xml:"r>t"`
			Text  string   `xml:"t"`
		} `xml:"si"`
	}
	if err := readXML(file, &data); err != nil {
		return nil, fmt.Errorf("read shared strings: %w", err)
	}

	values := make([]string, 0, len(data.Items))
	for _, item := range data.Items {
		if item.Text != "" {
			values = append(values, item.Text)
			continue
		}
		values = append(values, strings.Join(item.Texts, ""))
	}
	return values, nil
}

func readWorkbookSheets(files map[string]*zip.File) ([]sheetInfo, error) {
	var workbook workbookXML
	if err := readXML(files["xl/workbook.xml"], &workbook); err != nil {
		return nil, fmt.Errorf("read workbook: %w", err)
	}

	var rels relsXML
	if err := readXML(files["xl/_rels/workbook.xml.rels"], &rels); err != nil {
		return nil, fmt.Errorf("read workbook rels: %w", err)
	}

	targets := make(map[string]string, len(rels.Relationships))
	for _, rel := range rels.Relationships {
		target := strings.TrimPrefix(rel.Target, "/")
		if !strings.HasPrefix(target, "xl/") {
			target = path.Join("xl", target)
		}
		targets[rel.ID] = path.Clean(target)
	}

	sheets := make([]sheetInfo, 0, len(workbook.Sheets))
	for _, sheet := range workbook.Sheets {
		target, ok := targets[sheet.RID]
		if !ok {
			return nil, fmt.Errorf("missing relationship for sheet %s", sheet.Name)
		}
		sheets = append(sheets, sheetInfo{Name: sheet.Name, Path: target})
	}
	return sheets, nil
}

func readSheet(files map[string]*zip.File, filePath string, shared []string) (map[int]map[int]string, error) {
	file, ok := files[filePath]
	if !ok {
		return nil, fmt.Errorf("missing sheet file %s", filePath)
	}

	var data sheetXML
	if err := readXML(file, &data); err != nil {
		return nil, fmt.Errorf("read sheet %s: %w", filePath, err)
	}

	rows := make(map[int]map[int]string, len(data.Rows))
	for _, row := range data.Rows {
		cols := make(map[int]string, len(row.Cells))
		for _, cell := range row.Cells {
			col := columnIndex(cell.Ref)
			value := strings.TrimSpace(cell.value(shared))
			if value != "" {
				cols[col] = value
			}
		}
		if len(cols) > 0 {
			rows[row.R] = cols
		}
	}
	return rows, nil
}

func (c cellXML) value(shared []string) string {
	switch c.Type {
	case "s":
		index, err := strconv.Atoi(c.Value)
		if err == nil && index >= 0 && index < len(shared) {
			return shared[index]
		}
	case "inlineStr":
		return c.InlineStr.Text
	}
	return c.Value
}

func parseMonth(sheetName string, cells map[int]map[int]string) (Month, bool) {
	headers := cells[1]
	if len(headers) == 0 || !strings.EqualFold(headers[1], "Name") {
		return Month{}, false
	}

	dateColumns := make([]int, 0)
	dates := make([]string, 0)
	debtColumn := 0
	noteColumn := 0

	for col := 2; col <= maxColumn(headers); col++ {
		header := strings.TrimSpace(headers[col])
		if date, ok := parseDateHeader(header); ok {
			dateColumns = append(dateColumns, col)
			dates = append(dates, date)
			continue
		}
		normalized := strings.ToLower(header)
		if normalized == "debt" {
			debtColumn = col
		}
		if strings.Contains(normalized, "note") || strings.Contains(header, "หมายเหตุ") {
			noteColumn = col
		}
	}

	if len(dateColumns) == 0 {
		return Month{}, false
	}

	members := make([]Member, 0)
	for rowIndex := 2; ; rowIndex++ {
		row, ok := cells[rowIndex]
		if !ok {
			if rowIndex > maxRow(cells) {
				break
			}
			continue
		}

		name := strings.TrimSpace(row[1])
		if name == "" {
			continue
		}

		snapshots := make([]Snapshot, 0, len(dateColumns))
		for i, col := range dateColumns {
			value := parseInt(row[col])
			if value <= 0 {
				continue
			}
			snapshots = append(snapshots, Snapshot{
				Date: dates[i],
				Fans: value,
			})
		}

		member, ok := BuildMember(PlayerInput{
			Name:      name,
			Debt:      parseInt(row[debtColumn]),
			Note:      strings.TrimSpace(row[noteColumn]),
			Snapshots: snapshots,
		}, dates)
		if !ok {
			continue
		}

		members = append(members, member)
	}

	month := Month{
		ID:        sheetName,
		Label:     monthLabel(sheetName, dates[0]),
		StartDate: dates[0],
		EndDate:   dates[len(dates)-1],
		Dates:     dates,
		Members:   members,
	}
	return NormalizeMonth(month), true
}

func BuildMember(input PlayerInput, dates []string) (Member, bool) {
	name := strings.TrimSpace(input.Name)
	if name == "" || len(dates) == 0 {
		return Member{}, false
	}

	rawFans := make([]int64, len(dates))
	dateIndex := make(map[string]int, len(dates))
	for i, date := range dates {
		dateIndex[date] = i
	}

	hasAnySnapshot := false
	currentFans := int64(0)
	for _, snapshot := range input.Snapshots {
		index, ok := dateIndex[snapshot.Date]
		if !ok || snapshot.Fans <= 0 {
			continue
		}
		rawFans[index] = snapshot.Fans
		hasAnySnapshot = true
		if index == 0 || snapshot.Date >= dates[0] {
			currentFans = snapshot.Fans
		}
	}
	if !hasAnySnapshot {
		return Member{}, false
	}

	member := Member{
		Name:  name,
		Debt:  input.Debt,
		Note:  strings.TrimSpace(input.Note),
		Trend: "flat",
	}

	lastKnownFans := int64(0)
	for i, fans := range rawFans {
		if fans > 0 {
			lastKnownFans = fans
			currentFans = fans
		}
		member.Snapshots = append(member.Snapshots, Snapshot{
			Date: dates[i],
			Fans: lastKnownFans,
		})
	}

	member.CurrentFans = currentFans
	for i := 1; i < len(member.Snapshots); i++ {
		gain := int64(0)
		if rawFans[i] > 0 && member.Snapshots[i-1].Fans > 0 {
			gain = rawFans[i] - member.Snapshots[i-1].Fans
		}
		member.WeeklyGains = append(member.WeeklyGains, WeekGain{
			Index: i,
			Date:  member.Snapshots[i].Date,
			Gain:  gain,
		})
		member.TotalGain += gain
	}
	if len(member.WeeklyGains) > 0 {
		member.AverageGain = int64(math.Round(float64(member.TotalGain) / float64(len(member.WeeklyGains))))
		lastGain := member.WeeklyGains[len(member.WeeklyGains)-1].Gain
		switch {
		case lastGain > member.AverageGain:
			member.Trend = "up"
		case lastGain < member.AverageGain:
			member.Trend = "down"
		}
	}

	return member, true
}

func NormalizeMonth(month Month) Month {
	members := make([]Member, 0, len(month.Members))
	for _, member := range month.Members {
		rebuilt, ok := BuildMember(PlayerInput{
			Name:      member.Name,
			Debt:      member.Debt,
			Note:      member.Note,
			Snapshots: member.Snapshots,
		}, month.Dates)
		if ok {
			members = append(members, rebuilt)
		}
	}

	sort.SliceStable(members, func(i, j int) bool {
		return members[i].CurrentFans > members[j].CurrentFans
	})
	for index := range members {
		members[index].Rank = index + 1
	}

	month.Members = members
	month.Stats = buildStats(members)
	return month
}

func parseOldMembers(cells map[int]map[int]string) []OldMember {
	members := make([]OldMember, 0)
	for rowIndex := 1; rowIndex <= maxRow(cells); rowIndex++ {
		name := strings.TrimSpace(cells[rowIndex][1])
		if name != "" {
			members = append(members, OldMember{Name: name})
		}
	}
	return members
}

func buildStats(members []Member) MonthStats {
	var stats MonthStats
	stats.TrackedMembers = int64(len(members))

	for _, member := range members {
		stats.TotalCurrent += member.CurrentFans
		stats.TotalGain += member.TotalGain
		stats.TotalDebt += member.Debt
		if member.TotalGain > 0 {
			stats.ActiveMembers++
		}
		if stats.Leader == nil || member.CurrentFans > stats.Leader.Value {
			stats.Leader = &MemberMetric{Name: member.Name, Value: member.CurrentFans}
		}
		if stats.MostImproved == nil || member.TotalGain > stats.MostImproved.Value {
			stats.MostImproved = &MemberMetric{Name: member.Name, Value: member.TotalGain}
		}
	}

	if len(members) > 0 {
		stats.AverageGain = int64(math.Round(float64(stats.TotalGain) / float64(len(members))))
	}
	return stats
}

func readXML(file *zip.File, dest any) error {
	if file == nil {
		return fmt.Errorf("missing xml file")
	}
	handle, err := file.Open()
	if err != nil {
		return err
	}
	defer handle.Close()

	data, err := io.ReadAll(handle)
	if err != nil {
		return err
	}
	return xml.Unmarshal(data, dest)
}

func parseDateHeader(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}

	if number, err := strconv.ParseFloat(value, 64); err == nil && number > 30000 {
		return excelSerialToDate(number).Format("2006-01-02"), true
	}

	for _, layout := range []string{"2/1/2006", "02/01/2006", "1/2/2006", "01/02/2006"} {
		parsed, err := time.ParseInLocation(layout, value, time.UTC)
		if err == nil {
			return parsed.Format("2006-01-02"), true
		}
	}

	return "", false
}

func excelSerialToDate(serial float64) time.Time {
	wholeDays := int(serial)
	return time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC).AddDate(0, 0, wholeDays)
}

func parseInt(value string) int64 {
	value = strings.TrimSpace(strings.ReplaceAll(value, ",", ""))
	if value == "" || value == "=" {
		return 0
	}
	number, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return int64(math.Round(number))
}

func maxColumn(row map[int]string) int {
	max := 0
	for col := range row {
		if col > max {
			max = col
		}
	}
	return max
}

func maxRow(rows map[int]map[int]string) int {
	max := 0
	for row := range rows {
		if row > max {
			max = row
		}
	}
	return max
}

func columnIndex(ref string) int {
	index := 0
	for _, char := range ref {
		if char < 'A' || char > 'Z' {
			break
		}
		index = index*26 + int(char-'A'+1)
	}
	return index
}

func monthLabel(sheetName string, fallbackDate string) string {
	parsed, err := time.Parse("Jan2006", sheetName)
	if err == nil {
		return parsed.Format("January 2006")
	}

	date, err := time.Parse("2006-01-02", fallbackDate)
	if err == nil {
		return date.Format("January 2006")
	}
	return sheetName
}
