export interface Workbook {
  generatedAt: string;
  latestId: string;
  months: Month[];
  oldMembers: OldMember[];
  summary: BookSummary;
}

export interface BookSummary {
  monthCount: number;
  trackedMembers: number;
  totalCurrentFans: number;
}

export interface Month {
  id: string;
  label: string;
  startDate: string;
  endDate: string;
  dates: string[];
  stats: MonthStats;
  members: Member[];
}

export interface MonthInput {
  id: string;
  label: string;
  startDate: string;
  endDate: string;
  dates: string[];
  sourceMonthId: string;
}

export interface MonthStats {
  trackedMembers: number;
  activeMembers: number;
  totalCurrent: number;
  totalGain: number;
  averageGain: number;
  totalDebt: number;
  leader?: MemberMetric;
  mostImproved?: MemberMetric;
}

export interface MemberMetric {
  name: string;
  value: number;
}

export interface Member {
  rank: number;
  name: string;
  currentFans: number;
  totalGain: number;
  averageGain: number;
  debt: number;
  note?: string;
  trend: 'up' | 'down' | 'flat';
  snapshots: Snapshot[];
  weeklyGains: WeekGain[];
}

export interface Snapshot {
  date: string;
  fans: number;
}

export interface PlayerInput {
  name: string;
  debt: number;
  note: string;
  snapshots: Snapshot[];
}

export interface WeekGain {
  index: number;
  date: string;
  gain: number;
}

export interface OldMember {
  name: string;
}

export type SortMode = 'rank' | 'gain' | 'debt';
