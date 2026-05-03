import { Pipe, PipeTransform } from '@angular/core';

@Pipe({ name: 'compactNumber' })
export class CompactNumberPipe implements PipeTransform {
  transform(value: number | null | undefined): string {
    if (value === null || value === undefined || Number.isNaN(value)) {
      return '-';
    }

    const sign = value < 0 ? '-' : '';
    const absolute = Math.abs(value);
    if (absolute >= 1_000_000_000) {
      return `${sign}${this.trim(absolute / 1_000_000_000)}B`;
    }
    if (absolute >= 1_000_000) {
      return `${sign}${this.trim(absolute / 1_000_000)}M`;
    }
    if (absolute >= 1_000) {
      return `${sign}${this.trim(absolute / 1_000)}K`;
    }
    return `${sign}${absolute}`;
  }

  private trim(value: number): string {
    return value.toFixed(value >= 10 ? 1 : 2).replace(/\.0+$/, '').replace(/(\.\d)0$/, '$1');
  }
}
