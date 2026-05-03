import { Component, OnInit } from '@angular/core';

import { CrudAlertService } from '../core/crud-alert.service';
import { FanPointApiService } from '../core/fan-point-api.service';
import { Member, Month, SortMode, Workbook } from '../models/fan-point.models';

@Component({
  selector: 'app-dashboard',
  templateUrl: './dashboard.component.html',
  styleUrl: './dashboard.component.css'
})
export class DashboardComponent implements OnInit {
  workbook?: Workbook;
  selectedMonth?: Month;
  monthSearchTerm = '';
  isMonthAutocompleteOpen = false;
  selectedMember?: Member;
  searchTerm = '';
  noteSearchTerm = '';
  isMemberAutocompleteOpen = false;
  sortMode: SortMode = 'rank';
  isLoading = true;
  isSavingPlayer = false;
  isSavingMonth = false;
  errorMessage = '';
  viewMode: 'dashboard' | 'chart' = 'dashboard';

  readonly sortModes: Array<{ value: SortMode; label: string }> = [
    { value: 'rank', label: 'Rank' },
    { value: 'gain', label: 'Gain' },
    { value: 'debt', label: 'Debt' }
  ];

  constructor(
    private readonly api: FanPointApiService,
    private readonly alerts: CrudAlertService
  ) {}

  ngOnInit(): void {
    this.loadWorkbook();
  }

  switchView(mode: 'dashboard' | 'chart'): void {
    this.viewMode = mode;
  }

  exportSelectedMonth(): void {
    if (!this.selectedMonth) {
      return;
    }

    const month = this.selectedMonth;
    this.api.exportMonth(month.id).subscribe({
      next: (file) => {
        this.downloadBlob(file, `${this.safeFileName(month.id)}-summary.xlsx`);
      },
      error: (error: unknown) => {
        void this.alerts.error('Export failed', this.errorText(error));
      }
    });
  }

  loadWorkbook(preferredMonthId?: string, preferredMemberName?: string): void {
    if (!this.workbook) {
      this.isLoading = true;
    }

    this.api.getWorkbook().subscribe({
      next: (workbook) => {
        this.workbook = workbook;
        this.selectedMonth = preferredMonthId
          ? workbook.months.find((month) => month.id === preferredMonthId) ?? this.latestMonth(workbook)
          : this.latestMonth(workbook);
        this.monthSearchTerm = this.selectedMonth?.label ?? '';
        this.selectedMember = preferredMemberName
          ? this.selectedMonth?.members.find((member) => member.name === preferredMemberName) ?? this.selectedMonth?.members[0]
          : this.selectedMonth?.members[0];
        this.isLoading = false;
      },
      error: () => {
        this.errorMessage = 'Load data from backend failed';
        this.isLoading = false;
      }
    });
  }

  selectMonth(month: Month): void {
    this.selectedMonth = month;
    this.monthSearchTerm = month.label;
    this.isMonthAutocompleteOpen = false;
    this.selectedMember = month.members[0];
    this.searchTerm = '';
    this.noteSearchTerm = '';
  }

  openMonthAutocomplete(): void {
    this.isMonthAutocompleteOpen = true;
  }

  closeMonthAutocomplete(): void {
    window.setTimeout(() => {
      this.isMonthAutocompleteOpen = false;
      this.monthSearchTerm = this.selectedMonth?.label ?? '';
    }, 120);
  }

  filteredMonths(): Month[] {
    const months = this.workbook?.months ?? [];
    const query = this.monthSearchTerm.trim().toLowerCase();
    if (!query) {
      return months;
    }
    return months.filter((month) =>
      `${month.label} ${month.id}`.toLowerCase().includes(query)
    );
  }

  selectMember(member: Member): void {
    this.selectedMember = member;
    this.searchTerm = member.name;
    this.isMemberAutocompleteOpen = false;
  }

  openMemberAutocomplete(): void {
    this.isMemberAutocompleteOpen = true;
  }

  closeMemberAutocomplete(): void {
    window.setTimeout(() => {
      this.isMemberAutocompleteOpen = false;
    }, 120);
  }

  chooseMember(member: Member): void {
    this.selectMember(member);
  }

  async openCreateMonth(): Promise<void> {
    if (!this.workbook) {
      return;
    }
    const source = this.selectedMonth ?? this.latestMonth(this.workbook);
    const payload = await this.alerts.monthForm('create', {
      id: '',
      label: '',
      startDate: '',
      endDate: '',
      dates: source?.dates ?? [],
      sourceMonthId: source?.id ?? ''
    }, this.workbook.months);
    if (!payload) {
      return;
    }

    this.isSavingMonth = true;
    this.api.createMonth(payload).subscribe({
      next: (month) => {
        this.isSavingMonth = false;
        void this.alerts.success('Month created.', `${month.label} is ready.`);
        this.loadWorkbook(month.id);
      },
      error: (error: unknown) => {
        this.isSavingMonth = false;
        void this.alerts.error('Month change failed', this.errorText(error));
      }
    });
  }

  async openEditMonth(): Promise<void> {
    if (!this.selectedMonth || !this.workbook) {
      return;
    }
    const month = this.selectedMonth;
    const payload = await this.alerts.monthForm('edit', {
      id: month.id,
      label: month.label,
      startDate: month.startDate,
      endDate: month.endDate,
      dates: month.dates,
      sourceMonthId: ''
    }, this.workbook.months);
    if (!payload) {
      return;
    }

    this.isSavingMonth = true;
    this.api.updateMonth(month.id, payload).subscribe({
      next: (updatedMonth) => {
        this.isSavingMonth = false;
        void this.alerts.success('Month updated.', `${updatedMonth.label} is ready.`);
        this.loadWorkbook(updatedMonth.id);
      },
      error: (error: unknown) => {
        this.isSavingMonth = false;
        void this.alerts.error('Month change failed', this.errorText(error));
      }
    });
  }

  async deleteSelectedMonth(): Promise<void> {
    if (!this.selectedMonth || this.isSavingMonth) {
      return;
    }
    const month = this.selectedMonth;
    const confirmed = await this.alerts.confirm(
      'Delete month?',
      `Delete ${month.label} and all players in this month?`,
      'Delete Month'
    );
    if (!confirmed) {
      return;
    }

    this.isSavingMonth = true;
    this.api.deleteMonth(month.id).subscribe({
      next: () => {
        this.isSavingMonth = false;
        void this.alerts.success('Month deleted', `${month.label} was removed.`);
        this.loadWorkbook();
      },
      error: (error: unknown) => {
        this.isSavingMonth = false;
        void this.alerts.error('Delete month failed', this.errorText(error));
      }
    });
  }

  async openCreatePlayer(): Promise<void> {
    if (!this.selectedMonth) {
      return;
    }
    const month = this.selectedMonth;
    const payload = await this.alerts.playerForm('create', month.dates);
    if (!payload) {
      return;
    }

    this.isSavingPlayer = true;
    this.api.createPlayer(month.id, payload).subscribe({
      next: (player) => {
        this.isSavingPlayer = false;
        void this.alerts.success('Player created.', `${player.name} fan points were saved.`);
        this.loadWorkbook(month.id, player.name);
      },
      error: (error: unknown) => {
        this.isSavingPlayer = false;
        void this.alerts.error('Player change failed', this.errorText(error));
      }
    });
  }

  async openEditPlayer(): Promise<void> {
    if (!this.selectedMonth || !this.selectedMember) {
      return;
    }
    const month = this.selectedMonth;
    const originalName = this.selectedMember.name;
    const payload = await this.alerts.playerForm('edit', month.dates, this.selectedMember);
    if (!payload) {
      return;
    }

    this.isSavingPlayer = true;
    this.api.updatePlayer(month.id, originalName, payload).subscribe({
      next: (player) => {
        this.isSavingPlayer = false;
        void this.alerts.success('Player updated.', `${player.name} fan points were saved.`);
        this.loadWorkbook(month.id, player.name);
      },
      error: (error: unknown) => {
        this.isSavingPlayer = false;
        void this.alerts.error('Player change failed', this.errorText(error));
      }
    });
  }

  async deleteSelectedPlayer(): Promise<void> {
    if (!this.selectedMonth || !this.selectedMember || this.isSavingPlayer) {
      return;
    }
    const month = this.selectedMonth;
    const playerName = this.selectedMember.name;
    const confirmed = await this.alerts.confirm(
      'Delete player?',
      `Delete ${playerName} from ${month.label}?`,
      'Delete Player'
    );
    if (!confirmed) {
      return;
    }

    this.isSavingPlayer = true;
    this.api.deletePlayer(month.id, playerName).subscribe({
      next: () => {
        this.isSavingPlayer = false;
        void this.alerts.success('Player deleted', `${playerName} was removed from ${month.label}.`);
        this.loadWorkbook(month.id);
      },
      error: (error: unknown) => {
        this.isSavingPlayer = false;
        void this.alerts.error('Delete player failed', this.errorText(error));
      }
    });
  }

  filteredMembers(): Member[] {
    const members = [...(this.selectedMonth?.members ?? [])];
    const nameQuery = this.searchTerm.trim().toLowerCase();
    const noteQuery = this.noteSearchTerm.trim().toLowerCase();
    const filtered = members.filter((member) => {
      const matchesName = !nameQuery || member.name.toLowerCase().includes(nameQuery);
      const matchesNote = !noteQuery || (member.note ?? '').toLowerCase().includes(noteQuery);
      return matchesName && matchesNote;
    });

    switch (this.sortMode) {
      case 'gain':
        return filtered.sort((a, b) => b.totalGain - a.totalGain);
      case 'debt':
        return filtered.sort((a, b) => Math.abs(b.debt) - Math.abs(a.debt));
      default:
        return filtered.sort((a, b) => a.rank - b.rank);
    }
  }

  maxGain(member: Member | undefined): number {
    return Math.max(1, ...((member?.weeklyGains ?? []).map((week) => Math.abs(week.gain))));
  }

  weekBarWidth(gain: number, member: Member | undefined): string {
    return `${Math.max(4, Math.round((Math.abs(gain) / this.maxGain(member)) * 100))}%`;
  }

  trackByMonth(_: number, month: Month): string {
    return month.id;
  }

  trackByMember(_: number, member: Member): string {
    return member.name;
  }

  private latestMonth(workbook: Workbook): Month | undefined {
    return workbook.months.find((month) => month.id === workbook.latestId)
      ?? [...workbook.months].sort((a, b) => new Date(b.endDate).getTime() - new Date(a.endDate).getTime())[0];
  }

  private downloadBlob(blob: Blob, fileName: string): void {
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');

    link.href = url;
    link.download = fileName;
    link.click();
    URL.revokeObjectURL(url);
  }

  private safeFileName(value: string): string {
    return value.replace(/[^a-zA-Z0-9._-]+/g, '-').replace(/^-+|-+$/g, '') || 'month';
  }

  private errorText(error: unknown): string {
    if (typeof error === 'object' && error && 'error' in error) {
      const body = (error as { error?: unknown }).error;
      if (typeof body === 'object' && body && 'error' in body) {
        return String((body as { error: unknown }).error);
      }
      if (typeof body === 'string') {
        return body;
      }
    }
    return 'Change failed. Check MongoDB is enabled on the backend.';
  }
}
