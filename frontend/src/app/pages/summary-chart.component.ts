import { Component, Input, OnChanges, SimpleChanges, ViewChild, ElementRef, AfterViewInit } from '@angular/core';
import Chart from 'chart.js/auto';
import { Month } from '../models/fan-point.models';

@Component({
  selector: 'app-summary-chart',
  templateUrl: './summary-chart.component.html',
  styleUrl: './summary-chart.component.css'
})
export class SummaryChartComponent implements OnChanges, AfterViewInit {
  @Input() selectedMonth?: Month;
  @ViewChild('chartCanvas') canvasRef?: ElementRef<HTMLCanvasElement>;

  chart?: Chart;
  chartType: 'gain' | 'current' | 'average' = 'gain';

  ngAfterViewInit(): void {
    this.initChart();
  }

  ngOnChanges(changes: SimpleChanges): void {
    if (changes['selectedMonth'] && !changes['selectedMonth'].firstChange) {
      this.updateChart();
    }
  }

  initChart(): void {
    if (!this.canvasRef || !this.selectedMonth) {
      return;
    }
    this.createChart();
  }

  updateChart(): void {
    if (this.chart) {
      this.chart.destroy();
    }
    this.createChart();
  }

  changeChartType(type: 'gain' | 'current' | 'average'): void {
    this.chartType = type;
    this.updateChart();
  }

  private createChart(): void {
    if (!this.canvasRef || !this.selectedMonth) {
      return;
    }

    const canvas = this.canvasRef.nativeElement;
    const { labels, data, color } = this.getChartData();

    this.chart = new Chart(canvas, {
      type: 'bar',
      data: {
        labels,
        datasets: [
          {
            label: this.getChartLabel(),
            data,
            backgroundColor: color,
            borderColor: this.adjustBrightness(color, -20),
            borderWidth: 1,
            borderRadius: 4
          }
        ]
      },
      options: {
        responsive: true,
        maintainAspectRatio: true,
        indexAxis: 'y',
        plugins: {
          legend: {
            display: true,
            position: 'top'
          },
          title: {
            display: true,
            text: `${this.selectedMonth.label} - Player Summary`,
            font: { size: 16, weight: 'bold' }
          }
        },
        scales: {
          x: {
            beginAtZero: true,
            ticks: {
              callback: (value) => {
                if (typeof value === 'number') {
                  return value >= 1000 ? (value / 1000).toFixed(1) + 'K' : value.toString();
                }
                return value;
              }
            }
          }
        }
      }
    });
  }

  private getChartData(): { labels: string[]; data: number[]; color: string } {
    const members = (this.selectedMonth?.members || []).sort(
      (a, b) => b.currentFans - a.currentFans
    );

    let data: number[] = [];
    let color = '#3b82f6';

    switch (this.chartType) {
      case 'current':
        data = members.map((m) => m.currentFans);
        color = '#8b5cf6';
        break;
      case 'average':
        data = members.map((m) => m.averageGain);
        color = '#ec4899';
        break;
      case 'gain':
      default:
        data = members.map((m) => m.totalGain);
        color = '#10b981';
        break;
    }

    return {
      labels: members.map((m) => m.name),
      data,
      color
    };
  }

  private getChartLabel(): string {
    switch (this.chartType) {
      case 'current':
        return 'Current Fans';
      case 'average':
        return 'Avg Gain Per Week';
      case 'gain':
      default:
        return 'Total Gain';
    }
  }

  private adjustBrightness(color: string, percent: number): string {
    const usePound = color[0] === '#';
    const col = usePound ? color.slice(1) : color;
    const num = parseInt(col, 16);
    const amt = Math.round(2.55 * percent);
    const R = Math.min(255, Math.max(0, (num >> 16) + amt));
    const G = Math.min(255, Math.max(0, (num >> 8 & 0x00ff) + amt));
    const B = Math.min(255, Math.max(0, (num & 0x0000ff) + amt));
    return (usePound ? '#' : '') + (0x1000000 + (R < 256 ? R * 0x10000 : 0) + (G < 256 ? G * 0x100 : 0) + (B < 256 ? B : 0)).toString(16).slice(1);
  }
}
