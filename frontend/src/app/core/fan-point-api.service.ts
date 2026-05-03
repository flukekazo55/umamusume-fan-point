import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';

import { Member, Month, MonthInput, PlayerInput, Workbook } from '../models/fan-point.models';

@Injectable({ providedIn: 'root' })
export class FanPointApiService {
  constructor(private readonly http: HttpClient) {}

  getWorkbook(): Observable<Workbook> {
    return this.http.get<Workbook>('/api/months');
  }

  exportMonth(monthId: string): Observable<Blob> {
    return this.http.get(`/api/months/${encodeURIComponent(monthId)}/export`, {
      responseType: 'blob'
    });
  }

  createMonth(input: MonthInput): Observable<Month> {
    return this.http.post<Month>('/api/months', input);
  }

  updateMonth(monthId: string, input: MonthInput): Observable<Month> {
    return this.http.put<Month>(`/api/months/${encodeURIComponent(monthId)}`, input);
  }

  deleteMonth(monthId: string): Observable<void> {
    return this.http.delete<void>(`/api/months/${encodeURIComponent(monthId)}`);
  }

  createPlayer(monthId: string, input: PlayerInput): Observable<Member> {
    return this.http.post<Member>(`/api/months/${encodeURIComponent(monthId)}/players`, input);
  }

  updatePlayer(monthId: string, playerName: string, input: PlayerInput): Observable<Member> {
    return this.http.put<Member>(
      `/api/months/${encodeURIComponent(monthId)}/players/${encodeURIComponent(playerName)}`,
      input
    );
  }

  deletePlayer(monthId: string, playerName: string): Observable<void> {
    return this.http.delete<void>(
      `/api/months/${encodeURIComponent(monthId)}/players/${encodeURIComponent(playerName)}`
    );
  }
}
