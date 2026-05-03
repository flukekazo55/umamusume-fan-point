import { CommonModule } from '@angular/common';
import { HttpClientModule } from '@angular/common/http';
import { NgModule } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { BrowserModule } from '@angular/platform-browser';

import { AppComponent } from './app.component';
import { DashboardComponent } from './pages/dashboard.component';
import { SummaryChartComponent } from './pages/summary-chart.component';
import { CompactNumberPipe } from './shared/compact-number.pipe';

@NgModule({
  declarations: [AppComponent, DashboardComponent, SummaryChartComponent, CompactNumberPipe],
  imports: [BrowserModule, CommonModule, FormsModule, HttpClientModule],
  bootstrap: [AppComponent]
})
export class AppModule {}
