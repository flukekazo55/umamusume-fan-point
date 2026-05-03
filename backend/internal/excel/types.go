package excel

type Workbook struct {
	GeneratedAt string      `json:"generatedAt"`
	LatestID    string      `json:"latestId"`
	Months      []Month     `json:"months"`
	OldMembers  []OldMember `json:"oldMembers"`
	Summary     BookSummary `json:"summary"`
}

type BookSummary struct {
	MonthCount       int   `json:"monthCount"`
	TrackedMembers   int   `json:"trackedMembers"`
	TotalCurrentFans int64 `json:"totalCurrentFans"`
}

type Month struct {
	ID        string     `json:"id"`
	Label     string     `json:"label"`
	StartDate string     `json:"startDate"`
	EndDate   string     `json:"endDate"`
	Dates     []string   `json:"dates"`
	Stats     MonthStats `json:"stats"`
	Members   []Member   `json:"members"`
}

type MonthStats struct {
	TrackedMembers int64         `json:"trackedMembers"`
	ActiveMembers  int64         `json:"activeMembers"`
	TotalCurrent   int64         `json:"totalCurrent"`
	TotalGain      int64         `json:"totalGain"`
	AverageGain    int64         `json:"averageGain"`
	TotalDebt      int64         `json:"totalDebt"`
	Leader         *MemberMetric `json:"leader,omitempty"`
	MostImproved   *MemberMetric `json:"mostImproved,omitempty"`
}

type MemberMetric struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

type Member struct {
	Rank        int        `json:"rank"`
	Name        string     `json:"name"`
	CurrentFans int64      `json:"currentFans"`
	TotalGain   int64      `json:"totalGain"`
	AverageGain int64      `json:"averageGain"`
	Debt        int64      `json:"debt"`
	Note        string     `json:"note,omitempty"`
	Trend       string     `json:"trend"`
	Snapshots   []Snapshot `json:"snapshots"`
	WeeklyGains []WeekGain `json:"weeklyGains"`
}

type Snapshot struct {
	Date string `json:"date"`
	Fans int64  `json:"fans"`
}

type WeekGain struct {
	Index int    `json:"index"`
	Date  string `json:"date"`
	Gain  int64  `json:"gain"`
}

type OldMember struct {
	Name string `json:"name"`
}

type PlayerInput struct {
	Name      string     `json:"name"`
	Debt      int64      `json:"debt"`
	Note      string     `json:"note"`
	Snapshots []Snapshot `json:"snapshots"`
}

type MonthInput struct {
	ID            string   `json:"id"`
	Label         string   `json:"label"`
	StartDate     string   `json:"startDate"`
	EndDate       string   `json:"endDate"`
	Dates         []string `json:"dates"`
	SourceMonthID string   `json:"sourceMonthId"`
}
