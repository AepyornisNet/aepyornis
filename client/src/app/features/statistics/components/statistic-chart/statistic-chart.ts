import {
  AfterViewInit,
  ChangeDetectionStrategy,
  Component,
  effect,
  ElementRef,
  input,
  OnDestroy,
  viewChild,
} from '@angular/core';

import {
  BarController,
  BarElement,
  CategoryScale,
  Chart,
  ChartConfiguration,
  Colors,
  Legend,
  LinearScale,
  TimeScale,
  Tooltip,
} from 'chart.js';
import 'chartjs-adapter-date-fns';
import { Statistics } from '../../../../core/types/statistics';
import { UserPreferredUnits } from '../../../../core/types/user';

Chart.register(
  TimeScale,
  CategoryScale,
  LinearScale,
  BarController,
  BarElement,
  Colors,
  Tooltip,
  Legend,
);

@Component({
  selector: 'app-statistic-chart',
  imports: [],
  template: ` <canvas #chartCanvas></canvas> `,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class StatisticChartComponent implements AfterViewInit, OnDestroy {
  private readonly chartCanvas = viewChild<ElementRef<HTMLCanvasElement>>('chartCanvas');

  public readonly stats = input.required<Statistics | null>();
  public readonly preferredUnits = input<UserPreferredUnits>();
  public readonly filterNoDuration = input<boolean>(false);
  public readonly type = input.required<string>();
  public readonly unit = input<string>();

  private chart?: Chart;

  public constructor() {
    effect(() => {
      const statsData = this.stats();
      if (statsData && this.chart) {
        this.updateChart();
      }
    });
  }

  public ngAfterViewInit(): void {
    this.initChart();
  }

  public ngOnDestroy(): void {
    if (this.chart) {
      this.chart.destroy();
    }
  }

  private initChart(): void {
    const canvasRef = this.chartCanvas();
    if (!canvasRef) {
      return;
    }

    const ctx = canvasRef.nativeElement.getContext('2d');
    if (!ctx) {
      return;
    }

    const config: ChartConfiguration<'bar'> = {
      type: 'bar',
      data: {
        datasets: [],
      },
      options: {
        responsive: true,
        maintainAspectRatio: true,
        plugins: {
          legend: {
            position: 'top',
          },
          tooltip: {
            callbacks: {
              label: (context) => {
                const label = context.dataset.label || '';
                const value = context.parsed.y || 0;
                return this.formatTooltipValue(label, value);
              },
            },
          },
        },
        scales: {
          x: {
            type: 'time',
            time: {
              unit: 'month',
            },
          },
          y: {
            beginAtZero: true,
            ticks: {
              callback: (value) => this.formatYAxisValue(Number(value)),
            },
          },
        },
      },
    };

    this.chart = new Chart(ctx, config);
    this.updateChart();
  }

  private updateChart(): void {
    if (!this.chart) {
      return;
    }

    const statsData = this.stats();
    if (!statsData || !statsData.buckets) {
      this.chart.data.datasets = [];
      this.chart.update();
      return;
    }

    const datasets = Object.entries(statsData.buckets)
      .map(([, value]) => {
        const data = Object.values(value.buckets)
          .filter((e) => !this.filterNoDuration() || e.duration > 0)
          .map((e) => ({
            x: e.bucket,
            y: this.getValueForType(e),
          }));

        if (data.length === 0) {
          return null;
        }

        return {
          label: value.local_workout_type,
          data: data,
        };
      })
      .filter((dataset): dataset is NonNullable<typeof dataset> => dataset !== null);

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    this.chart.data.datasets = datasets as any;
    this.chart.update();
  }

  private getValueForType(bucket: Record<string, unknown>): number {
    const typeStr = this.type();
    const value = bucket[typeStr] as unknown;
    return typeof value === 'number' ? (value as number) : 0;
  }

  private formatTooltipValue(label: string, value: number): string {
    const unitType = this.unit();
    const preferredUnitsData = this.preferredUnits();

    if (unitType === 'duration') {
      return `${label}: ${this.formatDuration(value)}`;
    } else if (unitType && preferredUnitsData) {
      const unitValue = preferredUnitsData[unitType as keyof UserPreferredUnits];
      return `${label}: ${value} ${unitValue || ''}`;
    }
    return `${label}: ${value}`;
  }

  private formatYAxisValue(value: number): string {
    const unitType = this.unit();
    const preferredUnitsData = this.preferredUnits();

    if (unitType === 'duration') {
      return this.formatDuration(value);
    } else if (unitType && preferredUnitsData) {
      const unitValue = preferredUnitsData[unitType as keyof UserPreferredUnits];
      return `${value} ${unitValue || ''}`;
    }
    return value.toString();
  }

  private formatDuration(seconds: number): string {
    if (seconds < 0) {
      seconds = -seconds;
    }
    const time = {
      d: Math.floor(seconds / 86400),
      h: Math.floor(seconds / 3600) % 24,
      m: Math.floor(seconds / 60) % 60,
      s: Math.floor(seconds) % 60,
    };
    return Object.entries(time)
      .filter(([, val]) => val !== 0)
      .map(([key, val]) => `${val}${key}`)
      .join(' ');
  }
}
