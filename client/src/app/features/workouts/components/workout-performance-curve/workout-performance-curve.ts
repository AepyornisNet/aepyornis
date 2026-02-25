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
  CategoryScale,
  Chart,
  ChartConfiguration,
  ChartOptions,
  Colors,
  Decimation,
  Filler,
  Legend,
  LinearScale,
  LineController,
  LineElement,
  LogarithmicScale,
  PointElement,
  Tooltip,
} from 'chart.js';
import { MapDataDetails } from '../../../../core/types/workout';
import { DEFAULT_POWER_COLOR } from '../zone-colors';

Chart.register(
  CategoryScale,
  LinearScale,
  LogarithmicScale,
  PointElement,
  LineController,
  LineElement,
  Filler,
  Decimation,
  Colors,
  Tooltip,
  Legend,
);

@Component({
  selector: 'app-workout-performance-curve',
  templateUrl: './workout-performance-curve.html',
  styleUrl: './workout-performance-curve.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class WorkoutPerformanceCurveComponent implements AfterViewInit, OnDestroy {
  private readonly chartCanvas = viewChild<ElementRef<HTMLCanvasElement>>('chartCanvas');

  public readonly mapData = input<MapDataDetails | undefined>();

  private chart?: Chart;

  public constructor() {
    effect(() => {
      this.mapData();

      if (!this.chart) {
        return;
      }

      this.updateChart();
    });
  }

  public ngAfterViewInit(): void {
    this.initChart();
  }

  public ngOnDestroy(): void {
    this.chart?.destroy();
  }

  private initChart(): void {
    const canvasRef = this.chartCanvas();
    const points = this.performanceCurvePoints();
    if (!canvasRef || points.length === 0) {
      return;
    }

    const canvasContext = canvasRef.nativeElement.getContext('2d');
    if (!canvasContext) {
      return;
    }

    const config: ChartConfiguration<'line', number[], string> = {
      type: 'line',
      data: {
        labels: points.map((point) => this.formatDuration(point.x)),
        datasets: [
          {
            label: 'Power curve',
            data: points.map((point) => point.y),
            yAxisID: 'power',
            spanGaps: true,
            pointRadius: 0,
            borderWidth: 1.5,
            borderColor: DEFAULT_POWER_COLOR,
          },
        ],
      },
      options: this.chartOptions(),
    };

    this.chart = new Chart(canvasContext, config);
  }

  private updateChart(): void {
    const points = this.performanceCurvePoints();

    if (!this.chart) {
      if (points.length > 0) {
        this.initChart();
      }
      return;
    }

    if (points.length === 0) {
      this.chart.destroy();
      this.chart = undefined;
      return;
    }

    this.chart.data.datasets = [
      {
        type: 'line',
        label: 'Power curve',
        data: points.map((point) => point.y),
        yAxisID: 'power',
        spanGaps: true,
        pointRadius: 0,
        borderWidth: 1.5,
        borderColor: DEFAULT_POWER_COLOR,
      },
    ];
    this.chart.data.labels = points.map((point) => this.formatDuration(point.x));
    this.chart.options = this.chartOptions();
    this.chart.update();
  }

  private powerValues(): number[] {
    const values = this.mapData()?.extra_metrics?.['power'];
    if (!Array.isArray(values)) {
      return [];
    }

    return values.filter((value): value is number => typeof value === 'number' && Number.isFinite(value));
  }

  private performanceCurvePoints(): { x: number; y: number }[] {
    const power = this.powerValues();
    if (power.length < 2) {
      return [];
    }

    const sampleSeconds = this.estimateSampleSeconds();
    const prefixSum: number[] = new Array(power.length + 1).fill(0);

    for (let i = 0; i < power.length; i++) {
      prefixSum[i + 1] = prefixSum[i] + power[i];
    }

    const maxSeconds = Math.floor(power.length * sampleSeconds);
    const durations = this.durationCandidates(maxSeconds, sampleSeconds);
    const points: { x: number; y: number }[] = [];

    for (const duration of durations) {
      const windowSize = Math.max(1, Math.round(duration / sampleSeconds));
      if (windowSize > power.length) {
        break;
      }

      let bestAverage = 0;
      for (let start = 0; start + windowSize <= power.length; start++) {
        const windowSum = prefixSum[start + windowSize] - prefixSum[start];
        const avgPower = windowSum / windowSize;
        if (avgPower > bestAverage) {
          bestAverage = avgPower;
        }
      }

      const xSeconds = Math.round(windowSize * sampleSeconds);
      points.push({ x: xSeconds, y: bestAverage });
    }

    let currentMax = Number.POSITIVE_INFINITY;
    return points.map((point) => {
      currentMax = Math.min(currentMax, point.y);
      return { x: point.x, y: currentMax };
    });
  }

  private durationCandidates(maxSeconds: number, sampleSeconds: number): number[] {
    const baseCandidates = [
      1, 5, 10, 15, 20, 30, 45, 60, 90, 120, 180, 240, 300, 360, 420, 600, 900, 1200,
      1800, 2400, 3600, 5400, 7200,
    ];

    const minimum = Math.max(1, Math.round(sampleSeconds));
    const result = new Set<number>();

    for (const value of baseCandidates) {
      if (value >= minimum && value <= maxSeconds) {
        result.add(value);
      }
    }

    if (result.size === 0) {
      for (let value = minimum; value <= maxSeconds; value += minimum) {
        result.add(value);
      }
    }

    return Array.from(result).sort((a, b) => a - b);
  }

  private estimateSampleSeconds(): number {
    const durations = this.mapData()?.duration;
    if (Array.isArray(durations) && durations.length > 2) {
      const deltas: number[] = [];
      for (let i = 1; i < durations.length; i++) {
        const delta = durations[i] - durations[i - 1];
        if (Number.isFinite(delta) && delta > 0) {
          deltas.push(delta);
        }
      }

      if (deltas.length > 0) {
        return this.median(deltas);
      }
    }

    const times = this.mapData()?.time;
    if (Array.isArray(times) && times.length > 2) {
      const deltas: number[] = [];
      for (let i = 1; i < times.length; i++) {
        const delta =
          (new Date(times[i]).valueOf() - new Date(times[i - 1]).valueOf()) / 1000;
        if (Number.isFinite(delta) && delta > 0) {
          deltas.push(delta);
        }
      }

      if (deltas.length > 0) {
        return this.median(deltas);
      }
    }

    return 1;
  }

  private median(values: number[]): number {
    const sorted = [...values].sort((a, b) => a - b);
    const middle = Math.floor(sorted.length / 2);
    if (sorted.length % 2 === 0) {
      return (sorted[middle - 1] + sorted[middle]) / 2;
    }
    return sorted[middle];
  }

  private formatDuration(seconds: number): string {
    const rounded = Math.max(1, Math.round(seconds));
    const hours = Math.floor(rounded / 3600);
    const minutes = Math.floor((rounded % 3600) / 60);
    const secs = rounded % 60;

    if (hours > 0) {
      return `${hours}h ${minutes}m`;
    }

    if (minutes > 0) {
      return `${minutes}m ${secs}s`;
    }

    return `${secs}s`;
  }

  private chartOptions(): ChartOptions<'line'> {
    return {
      maintainAspectRatio: false,
      animation: false,
      scales: {
        x: {
          type: 'category',
        },
        power: {
          type: 'linear',
          min: 0,
          position: 'left',
          ticks: {
            callback: (value: string | number): string => `${Number(value).toFixed(0)} W`,
          },
        },
      },
      interaction: {
        mode: 'index',
        intersect: false,
      },
      plugins: {
        decimation: {
          enabled: true,
          algorithm: 'lttb',
        },
        legend: {
          display: false,
        },
        tooltip: {
          callbacks: {
            title: (items): string => {
              const firstItem = items[0];
              if (!firstItem) {
                return '';
              }

              return firstItem.label || '';
            },
            label: (item): string => {
              const value = item.parsed.y;
              return `Max sustained power: ${typeof value === 'number' && Number.isFinite(value) ? value.toFixed(0) : '-'} W`;
            },
          },
        },
      },
    };
  }
}
