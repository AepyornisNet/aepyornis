import {
  AfterViewInit,
  ChangeDetectionStrategy,
  Component,
  effect,
  ElementRef,
  inject,
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
import { TranslateService } from '@ngx-translate/core';
import { Statistics } from '../../../../core/types/statistics';
import { UserPreferredUnits } from '../../../../core/types/user';
import { getSportLabel } from '../../../../core/i18n/sport-labels';

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
  private readonly translate = inject(TranslateService);

  public readonly stats = input.required<Statistics | null>();
  public readonly preferredUnits = input<UserPreferredUnits>();
  public readonly filterNoDuration = input<boolean>(false);
  public readonly type = input.required<string>();
  public readonly unit = input<string>();

  private chart?: Chart;

  public constructor() {
    effect(() => {
      const statsData = this.stats();
      this.preferredUnits();

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
              unit: this.resolveTimeUnit(this.stats()?.bucket_format),
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

    const timeUnit = this.resolveTimeUnit(statsData.bucket_format);
    const xScale = this.chart.options.scales?.['x'];
    if (xScale && 'time' in xScale && xScale.time) {
      xScale.time.unit = timeUnit;
    }

    const datasets = Object.entries(statsData.buckets)
      .map(([, value]) => {
        const data = Object.values(value.buckets)
          .filter((e) => !this.filterNoDuration() || e.duration > 0)
          .map((e) => ({
            x: this.normalizeBucketValue(e.bucket, timeUnit),
            y: this.getValueForType(e),
          }));

        if (data.length === 0) {
          return null;
        }

        return {
          label: this.translate.instant(getSportLabel(value.local_workout_type)),
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
    const numericValue = typeof value === 'number' ? (value as number) : 0;

    const unitType = this.unit();

    if (unitType === 'distance') {
      return this.convertDistance(numericValue);
    }

    if (unitType === 'speed') {
      return this.convertSpeed(numericValue);
    }

    return numericValue;
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

  private resolveTimeUnit(bucketFormat?: string): 'day' | 'week' | 'month' | 'year' {
    if (!bucketFormat) {
      return 'month';
    }

    const format = bucketFormat.toLowerCase();

    if (format.includes('w')) {
      return 'week';
    }

    if (format.includes('d')) {
      return 'day';
    }

    if (format.includes('m')) {
      return 'month';
    }

    return 'year';
  }

  private normalizeBucketValue(bucket: string, timeUnit: 'day' | 'week' | 'month' | 'year'): Date | string {
    const date = new Date(bucket);

    if (Number.isNaN(date.getTime())) {
      return bucket;
    }

    const normalized = new Date(date);

    if (timeUnit === 'week') {
      const day = normalized.getDay();
      const diffToMonday = (day + 6) % 7;
      normalized.setDate(normalized.getDate() - diffToMonday);
    } else if (timeUnit === 'month') {
      normalized.setDate(1);
    } else if (timeUnit === 'year') {
      normalized.setMonth(0, 1);
    }

    return normalized;
  }

  private convertDistance(value: number): number {
    const preferredUnits = this.preferredUnits();
    const distanceUnit = preferredUnits?.distance;

    if (distanceUnit === 'mi') {
      return value / 1609.344;
    }

    if (distanceUnit === 'm') {
      return value;
    }

    // Default to kilometers
    return value / 1000;
  }

  private convertSpeed(value: number): number {
    const preferredUnits = this.preferredUnits();
    const speedUnit = preferredUnits?.speed;

    if (speedUnit === 'mph') {
      return value * 2.23694;
    }

    // Default to km/h
    return value * 3.6;
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
