import { AfterViewInit, Component, ElementRef, Input, OnChanges, OnDestroy, SimpleChanges, ViewChild } from '@angular/core';
import Chart from 'chart.js/auto';
import { Member, Month } from '../models/fan-point.models';

type ChartMetric = 'gain' | 'current' | 'average';

interface ChartMetricConfig {
  label: string;
  color: string;
  borderColor: string;
  getValue: (member: Member) => number;
}

const CHART_METRICS: Record<ChartMetric, ChartMetricConfig> = {
  gain: {
    label: 'Total Gain',
    color: '#10b981',
    borderColor: '#0d946d',
    getValue: (member) => member.totalGain
  },
  current: {
    label: 'Current Fans',
    color: '#8b5cf6',
    borderColor: '#7443d3',
    getValue: (member) => member.currentFans
  },
  average: {
    label: 'Avg Gain Per Week',
    color: '#ec4899',
    borderColor: '#c93579',
    getValue: (member) => member.averageGain
  }
};

@Component({
  selector: 'app-summary-chart',
  templateUrl: './summary-chart.component.html',
  styleUrl: './summary-chart.component.css'
})
export class SummaryChartComponent implements OnChanges, AfterViewInit, OnDestroy {
  @Input() selectedMonth?: Month;
  @ViewChild('chartCanvas') canvasRef?: ElementRef<HTMLCanvasElement>;

  chartType: ChartMetric = 'gain';

  private chart?: Chart;

  ngAfterViewInit(): void {
    this.renderChart();
  }

  ngOnChanges(changes: SimpleChanges): void {
    if (changes['selectedMonth']) {
      this.renderChart();
    }
  }

  ngOnDestroy(): void {
    this.destroyChart();
  }

  changeChartType(type: ChartMetric): void {
    this.chartType = type;
    this.renderChart();
  }

  private renderChart(): void {
    if (!this.canvasRef || !this.selectedMonth) {
      return;
    }

    this.destroyChart();

    const metric = CHART_METRICS[this.chartType];
    const chartData = this.buildChartData(metric);

    this.chart = new Chart(this.canvasRef.nativeElement, {
      type: 'bar',
      data: {
        labels: chartData.labels,
        datasets: [
          {
            label: metric.label,
            data: chartData.data,
            backgroundColor: metric.color,
            borderColor: metric.borderColor,
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

  private buildChartData(metric: ChartMetricConfig): { labels: string[]; data: number[] } {
    const members = this.getMembersByCurrentFans();
    return {
      labels: members.map((member) => member.name),
      data: members.map((member) => metric.getValue(member))
    };
  }

  private getMembersByCurrentFans(): Member[] {
    return [...(this.selectedMonth?.members ?? [])].sort(
      (first, second) => second.currentFans - first.currentFans
    );
  }

  private destroyChart(): void {
    this.chart?.destroy();
    this.chart = undefined;
  }
}
