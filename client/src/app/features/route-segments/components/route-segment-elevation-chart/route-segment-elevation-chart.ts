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
  CategoryScale,
  Chart,
  ChartConfiguration,
  ChartDataset,
  Filler,
  LinearScale,
  LineController,
  LineElement,
  PointElement,
  ScriptableLineSegmentContext,
  Tooltip,
} from 'chart.js';
import { User } from '../../../../core/services/user';
import { metersToMiles } from '../../../../core/config/units';

Chart.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineController,
  LineElement,
  Filler,
  Tooltip,
);

export type ElevationPoint = {
  distance: number; // in km
  elevation: number; // in meters
};

@Component({
  selector: 'app-route-segment-elevation-chart',
  templateUrl: './route-segment-elevation-chart.html',
  styleUrl: './route-segment-elevation-chart.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class RouteSegmentElevationChartComponent implements AfterViewInit, OnDestroy {
  private readonly chartCanvas = viewChild<ElementRef<HTMLCanvasElement>>('chartCanvas');
  private readonly user = inject(User);

  public readonly points = input<ElevationPoint[]>([]);
  public readonly selection = input<{ startIndex: number; endIndex: number } | null>(null);

  private chart?: Chart<'line', { x: number; y: number }[]>;

  private get distanceUnit(): string {
    return this.user.getUserInfo()()?.profile?.profile.preferred_units.distance ?? 'km';
  }

  private get elevationUnit(): string {
    return this.user.getUserInfo()()?.profile?.profile.preferred_units.elevation ?? 'm';
  }

  public constructor() {
    effect(() => {
      // Re-render dataset when points or selection changes
      const pts = this.points();
      const sel = this.selection();
      if (!this.chart || pts.length === 0) {
        return;
      }
      this.updateChart(pts, sel);
    });
  }

  public ngAfterViewInit(): void {
    setTimeout(() => {
      this.initChart();
    }, 50);
  }

  public ngOnDestroy(): void {
    if (this.chart) {
      this.chart.destroy();
      this.chart = undefined;
    }
  }

  private convertDistance(km: number): number {
    if (this.distanceUnit === 'mi') {
      return km * 1000 * metersToMiles;
    }
    return km;
  }

  private convertElevation(meters: number): number {
    if (this.elevationUnit === 'ft') {
      return meters * 3.28084;
    }
    return meters;
  }

  private calculateBounds(
    pts: ElevationPoint[],
    sel: { startIndex: number; endIndex: number } | null,
  ): { xMin: number; xMax: number; yMin?: number; yMax?: number } {
    const totalPts = pts.length;
    const totalStartDist = this.convertDistance(pts[0].distance);
    const totalEndDist = this.convertDistance(pts[totalPts - 1].distance);

    let xMin = totalStartDist;
    let xMax = totalEndDist;

    if (sel && sel.startIndex >= 0 && sel.endIndex < totalPts) {
      const startDist = this.convertDistance(pts[sel.startIndex].distance);
      const endDist = this.convertDistance(pts[sel.endIndex].distance);
      const span = Math.max(0.05, endDist - startDist);
      const padding = span * 0.15; // 15% padding on each side
      xMin = Math.max(totalStartDist, startDist - padding);
      xMax = Math.min(totalEndDist, endDist + padding);
    }

    if (xMax <= xMin) {
      xMax = xMin + 0.1;
    }

    // Calculate Y bounds from points in the visible X window
    const visiblePts = pts.filter((p) => {
      const d = this.convertDistance(p.distance);
      return d >= xMin && d <= xMax;
    });

    let yMin: number | undefined;
    let yMax: number | undefined;

    if (visiblePts.length > 0) {
      let minElev = Infinity;
      let maxElev = -Infinity;
      for (const p of visiblePts) {
        const elev = this.convertElevation(p.elevation);
        if (elev < minElev) {
          minElev = elev;
        }
        if (elev > maxElev) {
          maxElev = elev;
        }
      }
      const elevSpan = Math.max(10, maxElev - minElev);
      const yPadding = elevSpan * 0.15;
      yMin = Math.floor(minElev - yPadding);
      yMax = Math.ceil(maxElev + yPadding);
    }

    return { xMin, xMax, yMin, yMax };
  }

  private initChart(): void {
    const canvasRef = this.chartCanvas();
    if (!canvasRef) {
      return;
    }

    const pts = this.points();
    if (pts.length === 0) {
      return;
    }

    const sel = this.selection();
    const isMiles = this.distanceUnit === 'mi';
    const isFeet = this.elevationUnit === 'ft';

    const distUnitLabel = isMiles ? 'mi' : 'km';
    const elevUnitLabel = isFeet ? 'ft' : 'm';

    const data = pts.map((p) => ({
      x: this.convertDistance(p.distance),
      y: Math.round(this.convertElevation(p.elevation)),
    }));

    const start = sel?.startIndex ?? -1;
    const end = sel?.endIndex ?? -1;
    const bounds = this.calculateBounds(pts, sel);

    const config: ChartConfiguration<'line', { x: number; y: number }[]> = {
      type: 'line',
      data: {
        datasets: [
          {
            label: 'Elevation',
            data,
            fill: true,
            tension: 0.15,
            borderWidth: 2,
            pointRadius: 0,
            pointHoverRadius: 4,
            segment: {
              borderColor: (ctx: ScriptableLineSegmentContext): string => {
                if (start >= 0 && end >= 0 && ctx.p0DataIndex >= start && ctx.p1DataIndex <= end) {
                  return '#0d6efd'; // Bootstrap Primary
                }
                return '#adb5bd'; // Muted Gray
              },
              backgroundColor: (ctx: ScriptableLineSegmentContext): string => {
                if (start >= 0 && end >= 0 && ctx.p0DataIndex >= start && ctx.p1DataIndex <= end) {
                  return 'rgba(13, 110, 253, 0.25)'; // Highlight area
                }
                return 'rgba(173, 181, 189, 0.08)'; // Dimmed area
              },
            },
          },
        ],
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        animation: false,
        interaction: {
          mode: 'index',
          intersect: false,
        },
        plugins: {
          legend: {
            display: false,
          },
          tooltip: {
            callbacks: {
              title: (items) => {
                if (!items.length) {
                  return '';
                }
                const xVal = items[0].parsed.x;
                return `${Number(xVal ?? 0).toFixed(2)} ${distUnitLabel}`;
              },
              label: (item) => {
                const yVal = item.parsed.y;
                return `${Math.round(yVal ?? 0)} ${elevUnitLabel}`;
              },
            },
          },
        },
        scales: {
          x: {
            type: 'linear',
            display: true,
            min: bounds.xMin,
            max: bounds.xMax,
            grid: {
              display: false,
            },
            ticks: {
              maxTicksLimit: 8,
              callback: (val) => `${Number(val).toFixed(2)} ${distUnitLabel}`,
            },
          },
          y: {
            display: true,
            min: bounds.yMin,
            max: bounds.yMax,
            grid: {
              color: 'rgba(0, 0, 0, 0.06)',
            },
            ticks: {
              maxTicksLimit: 4,
              callback: (val) => `${val} ${elevUnitLabel}`,
            },
          },
        },
      },
    };

    this.chart = new Chart(canvasRef.nativeElement, config);
  }

  private updateChart(
    pts: ElevationPoint[],
    sel: { startIndex: number; endIndex: number } | null,
  ): void {
    if (!this.chart) {
      this.initChart();
      return;
    }

    const data = pts.map((p) => ({
      x: this.convertDistance(p.distance),
      y: Math.round(this.convertElevation(p.elevation)),
    }));

    const start = sel?.startIndex ?? -1;
    const end = sel?.endIndex ?? -1;
    const bounds = this.calculateBounds(pts, sel);

    if (this.chart.options.scales?.['x']) {
      this.chart.options.scales['x'].min = bounds.xMin;
      this.chart.options.scales['x'].max = bounds.xMax;
    }
    if (this.chart.options.scales?.['y']) {
      this.chart.options.scales['y'].min = bounds.yMin;
      this.chart.options.scales['y'].max = bounds.yMax;
    }

    if (this.chart.data.datasets.length > 0) {
      const lineDataset = this.chart.data.datasets[0] as ChartDataset<
        'line',
        { x: number; y: number }[]
      >;
      lineDataset.data = data;
      lineDataset.segment = {
        borderColor: (ctx: ScriptableLineSegmentContext): string => {
          if (start >= 0 && end >= 0 && ctx.p0DataIndex >= start && ctx.p1DataIndex <= end) {
            return '#0d6efd';
          }
          return '#adb5bd';
        },
        backgroundColor: (ctx: ScriptableLineSegmentContext): string => {
          if (start >= 0 && end >= 0 && ctx.p0DataIndex >= start && ctx.p1DataIndex <= end) {
            return 'rgba(13, 110, 253, 0.25)';
          }
          return 'rgba(173, 181, 189, 0.08)';
        },
      };
    }

    this.chart.update('none');
  }
}
