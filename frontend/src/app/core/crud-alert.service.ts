import { Injectable } from '@angular/core';
import flatpickr from 'flatpickr';
import Swal, { SweetAlertIcon } from 'sweetalert2';

import { Member, Month, MonthInput, PlayerInput } from '../models/fan-point.models';

@Injectable({ providedIn: 'root' })
export class CrudAlertService {
  success(title: string, text?: string): Promise<void> {
    return this.show('success', title, text);
  }

  error(title: string, text?: string): Promise<void> {
    return this.show('error', title, text);
  }

  async confirm(title: string, text: string, confirmButtonText = 'Delete'): Promise<boolean> {
    const result = await Swal.fire({
      title,
      text,
      icon: 'warning',
      showCancelButton: true,
      confirmButtonText,
      cancelButtonText: 'Cancel',
      confirmButtonColor: '#b91c1c',
      cancelButtonColor: '#6b7280',
      reverseButtons: true
    });
    return result.isConfirmed;
  }

  async monthForm(mode: 'create' | 'edit', value: MonthInput, months: Month[]): Promise<MonthInput | undefined> {
    const isCreate = mode === 'create';
    const result = await Swal.fire<MonthInput>({
      title: isCreate ? 'Add Month' : 'Edit Month',
      html: this.monthFormHtml(isCreate, value, months),
      width: 720,
      showCancelButton: true,
      confirmButtonText: isCreate ? 'Create Month' : 'Save Month',
      cancelButtonText: 'Cancel',
      confirmButtonColor: '#e11d48',
      focusConfirm: false,
      customClass: { popup: 'crud-swal' },
      didOpen: () => this.bindMonthDatePickers(),
      preConfirm: () => {
        const formValue = this.readMonthForm(isCreate, value.id);
        if (!formValue.id && isCreate) {
          Swal.showValidationMessage('Month id is required.');
          return false;
        }
        if (!formValue.label) {
          Swal.showValidationMessage('Month label is required.');
          return false;
        }
        if (formValue.dates.length === 0) {
          Swal.showValidationMessage('Add at least one date.');
          return false;
        }
        return formValue;
      }
    });
    return result.isConfirmed ? result.value : undefined;
  }

  async playerForm(mode: 'create' | 'edit', dates: string[], member?: Member): Promise<PlayerInput | undefined> {
    const isCreate = mode === 'create';
    const result = await Swal.fire<PlayerInput>({
      title: isCreate ? 'Add Player' : 'Edit Player',
      html: this.playerFormHtml(dates, member),
      width: 760,
      showCancelButton: true,
      confirmButtonText: isCreate ? 'Create Player' : 'Save Player',
      cancelButtonText: 'Cancel',
      confirmButtonColor: '#e11d48',
      focusConfirm: false,
      customClass: { popup: 'crud-swal' },
      didOpen: () => this.bindNumberFormatting(),
      preConfirm: () => {
        const formValue = this.readPlayerForm(dates);
        if (!formValue.name) {
          Swal.showValidationMessage('Player name is required.');
          return false;
        }
        if (!formValue.snapshots.some((snapshot) => snapshot.fans > 0)) {
          Swal.showValidationMessage('Add at least one fan point value.');
          return false;
        }
        return formValue;
      }
    });
    return result.isConfirmed ? result.value : undefined;
  }

  private async show(icon: SweetAlertIcon, title: string, text?: string): Promise<void> {
    await Swal.fire({
      title,
      text,
      icon,
      confirmButtonText: 'OK',
      confirmButtonColor: icon === 'error' ? '#b91c1c' : '#e11d48'
    });
  }

  private monthFormHtml(isCreate: boolean, value: MonthInput, months: Month[]): string {
    const sourceOptions = [
      '<option value="">Blank month</option>',
      ...months.map((month) => {
        const selected = month.id === value.sourceMonthId ? ' selected' : '';
        return `<option value="${this.escape(month.id)}"${selected}>${this.escape(month.label)}</option>`;
      })
    ].join('');

    return `
      <div class="crud-form">
        <div class="crud-grid month-grid">
          <label>
            <span>ID</span>
            <input id="month-id" value="${this.escape(value.id)}" ${isCreate ? '' : 'disabled'}>
          </label>
          <label>
            <span>Label</span>
            <input id="month-label" value="${this.escape(value.label)}">
          </label>
          <label>
            <span>Start</span>
            <input id="month-start" data-flatpickr value="${this.escape(value.startDate)}">
          </label>
          <label>
            <span>End</span>
            <input id="month-end" data-flatpickr value="${this.escape(value.endDate)}">
          </label>
        </div>
        ${isCreate ? `
          <label>
            <span>Clone players from</span>
            <select id="month-source">${sourceOptions}</select>
          </label>
        ` : ''}
        <div class="crud-date-section">
          <div class="crud-section-header">
            <span>Dates</span>
            <button type="button" class="crud-add-button" id="month-add-date" title="Add date" aria-label="Add date">
              ${this.icon('plus')}
            </button>
          </div>
          <div class="crud-date-list" id="month-date-list">
            ${this.monthDatesHtml(value.dates)}
          </div>
        </div>
      </div>
    `;
  }

  private monthDatesHtml(dates: string[]): string {
    const values = dates.length > 0 ? dates : [''];
    return values.map((date) => this.monthDateRowHtml(date)).join('');
  }

  private monthDateRowHtml(date: string): string {
    return `
      <div class="crud-date-row">
        <input data-month-date data-flatpickr value="${this.escape(date)}">
        <button type="button" class="crud-remove-button" data-remove-date title="Remove date" aria-label="Remove date">
          ${this.icon('trash')}
        </button>
      </div>
    `;
  }

  private playerFormHtml(dates: string[], member?: Member): string {
    const snapshotByDate = new Map((member?.snapshots ?? []).map((snapshot) => [snapshot.date, snapshot.fans]));
    const pointInputs = dates.map((date) => `
      <label>
        <span>${this.escape(this.shortDate(date))}</span>
        <input
          id="fans-${this.escape(date)}"
          inputmode="numeric"
          data-number-input
          value="${this.escape(this.formatNumber(snapshotByDate.get(date) ?? 0))}">
      </label>
    `).join('');

    return `
      <div class="crud-form">
        <div class="crud-grid player-grid">
          <label>
            <span>Name</span>
            <input id="player-name" value="${this.escape(member?.name ?? '')}">
          </label>
          <label>
            <span>Debt</span>
            <input id="player-debt" inputmode="numeric" data-number-input value="${this.escape(this.formatNumber(member?.debt ?? 0))}">
          </label>
        </div>
        <label>
          <span>Note</span>
          <textarea id="player-note" rows="2">${this.escape(member?.note ?? '')}</textarea>
        </label>
        <div class="crud-grid points-grid">${pointInputs}</div>
      </div>
    `;
  }

  private readMonthForm(isCreate: boolean, fallbackID: string): MonthInput {
    const popup = Swal.getPopup();
    const dates = Array.from(popup?.querySelectorAll<HTMLInputElement>('[data-month-date]') ?? [])
      .map((input) => input.value.trim())
      .filter((date, index, values) => date && values.indexOf(date) === index)
      .sort();

    return {
      id: isCreate ? this.value('month-id') : fallbackID,
      label: this.value('month-label'),
      startDate: this.value('month-start') || dates[0] || '',
      endDate: this.value('month-end') || dates[dates.length - 1] || '',
      dates,
      sourceMonthId: isCreate ? this.value('month-source') : ''
    };
  }

  private readPlayerForm(dates: string[]): PlayerInput {
    return {
      name: this.value('player-name'),
      debt: this.parseNumber(this.value('player-debt')),
      note: this.value('player-note'),
      snapshots: dates
        .map((date) => ({ date, fans: this.parseNumber(this.value(`fans-${date}`)) }))
        .filter((snapshot) => snapshot.fans > 0)
    };
  }

  private bindNumberFormatting(): void {
    const popup = Swal.getPopup();
    popup?.querySelectorAll<HTMLInputElement>('[data-number-input]').forEach((input) => {
      input.addEventListener('blur', () => {
        input.value = this.formatNumber(input.value);
      });
    });
  }

  private bindMonthDatePickers(): void {
    const popup = Swal.getPopup();
    popup?.querySelectorAll<HTMLInputElement>('[data-flatpickr]').forEach((input) => {
      this.initializeDatePicker(input);
    });
    popup?.querySelectorAll<HTMLButtonElement>('[data-remove-date]').forEach((button) => {
      this.bindRemoveDate(button);
    });
    popup?.querySelector<HTMLButtonElement>('#month-add-date')?.addEventListener('click', () => {
      const list = popup.querySelector<HTMLDivElement>('#month-date-list');
      if (!list) {
        return;
      }
      const wrapper = document.createElement('div');
      wrapper.innerHTML = this.monthDateRowHtml('');
      const row = wrapper.firstElementChild;
      if (!row) {
        return;
      }
      list.appendChild(row);
      const input = row.querySelector<HTMLInputElement>('[data-month-date]');
      if (input) {
        this.initializeDatePicker(input);
        input.focus();
      }
      const removeButton = row.querySelector<HTMLButtonElement>('[data-remove-date]');
      if (removeButton) {
        this.bindRemoveDate(removeButton);
      }
    });
  }

  private initializeDatePicker(input: HTMLInputElement): void {
    if (input.dataset['flatpickrReady'] === 'true') {
      return;
    }
    flatpickr(input, {
      allowInput: true,
      dateFormat: 'Y-m-d'
    });
    input.dataset['flatpickrReady'] = 'true';
  }

  private bindRemoveDate(button: HTMLButtonElement): void {
    if (button.dataset['removeReady'] === 'true') {
      return;
    }
    button.addEventListener('click', () => {
      const row = button.closest('.crud-date-row');
      const list = button.closest('.crud-date-list');
      if (row && list && list.querySelectorAll('.crud-date-row').length > 1) {
        row.remove();
        return;
      }
      const input = row?.querySelector<HTMLInputElement>('[data-month-date]');
      if (input) {
        input.value = '';
        input.focus();
      }
    });
    button.dataset['removeReady'] = 'true';
  }

  private value(id: string): string {
    const popup = Swal.getPopup();
    const field = popup?.querySelector<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>(`#${CSS.escape(id)}`);
    return field?.value.trim() ?? '';
  }

  private parseNumber(value: string | number | null | undefined): number {
    const normalized = String(value ?? '').replace(/,/g, '').trim();
    if (!normalized || normalized === '-') {
      return 0;
    }
    const parsed = Number(normalized);
    return Number.isFinite(parsed) ? Math.round(parsed) : 0;
  }

  private formatNumber(value: string | number | null | undefined): string {
    const parsed = this.parseNumber(value);
    return parsed === 0 ? '' : parsed.toLocaleString('en-US');
  }

  private shortDate(value: string): string {
    const date = new Date(`${value}T00:00:00`);
    if (Number.isNaN(date.getTime())) {
      return value;
    }
    return date.toLocaleDateString('en-US', { day: 'numeric', month: 'short' });
  }

  private escape(value: string | number | null | undefined): string {
    return String(value ?? '')
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;');
  }

  private icon(name: 'plus' | 'trash'): string {
    if (name === 'plus') {
      return '<i class="fi fi-rr-plus" aria-hidden="true"></i>';
    }
    return '<i class="fi fi-rr-trash" aria-hidden="true"></i>';
  }
}
