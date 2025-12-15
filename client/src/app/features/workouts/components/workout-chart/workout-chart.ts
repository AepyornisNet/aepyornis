import {
  AfterViewInit,
  ChangeDetectionStrategy,
  Component,
  computed,
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
  ChartOptions,
  Colors,
  Decimation,
  Filler,
  Legend,
  LinearScale,
  LineController,
  LineElement,
  PointElement,
  TimeScale,
  Tooltip,
  TooltipItem,
} from 'chart.js';
import 'chartjs-adapter-date-fns';
import zoomPlugin from 'chartjs-plugin-zoom';
import { MapDataDetails } from '../../../../core/types/workout';
import { WorkoutDetailCoordinatorService } from '../../services/workout-detail-coordinator.service';
import { TranslatePipe } from '@ngx-translate/core';
import { User } from '../../../../core/services/user';
import {
  DEFAULT_HEART_RATE_COLOR,
  DEFAULT_POWER_COLOR,
  FTP_ZONE_COLORS,
  HR_ZONE_COLORS,
} from '../zone-colors';

Chart.register(
  TimeScale,
  CategoryScale,
  LinearScale,
  PointElement,
  LineController,
  LineElement,
  Filler,
  Decimation,
  Colors,
  Tooltip,
  Legend,
  zoomPlugin,
);

type MetricConfig = {
  formatter?: (val: number) => string;
  labelFormatter?: (val: number) => string;
  formatterYaxis?: boolean;
  yaxis?: boolean | { min?: number; max?: number; position?: string };
  hiddenByDefault?: boolean;
};

@Component({
  selector: 'app-workout-chart',
  imports: [TranslatePipe],
  templateUrl: './workout-chart.html',
  styleUrl: './workout-chart.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class WorkoutChartComponent implements AfterViewInit, OnDestroy {
  private readonly chartCanvas = viewChild<ElementRef<HTMLCanvasElement>>('chartCanvas');

  public readonly mapData = input<MapDataDetails | undefined>();
  public readonly extraMetrics = input<string[]>([]);

  private coordinatorService = inject(WorkoutDetailCoordinatorService);
  private userService = inject(User);
  private chart?: Chart;
  private timeLabels: number[] = [];
  private isUpdatingFromZoom = false; // Flag to prevent infinite loops
  public viewMode: 'time' | 'distance' = 'time';

  private readonly speedUnit = computed(() =>
    this.userService.getUserInfo()()?.profile?.preferred_units?.speed || 'km/h',
  );

  private get mapDataValue(): MapDataDetails | undefined {
    return this.mapData();
  }

  private get extraMetricsValue(): string[] {
    return this.extraMetrics();
  }

  public constructor() {
    // React to interval selection changes from the coordinator service
    effect(() => {
      const selection = this.coordinatorService.selectedInterval();
      const mapData = this.mapDataValue;

      // Don't react if the change came from our own zoom
      if (this.isUpdatingFromZoom) {
        return;
      }

      if (!mapData || !selection) {
        // Only reset zoom if there's no selection
        if (!selection && this.chart) {
          this.resetZoom();
        }
        return;
      }

      // Zoom to the selected interval
      const startTime = new Date(mapData.time[selection.startIndex]).getTime();
      const endTime = new Date(mapData.time[selection.endIndex]).getTime();
      this.zoomToRange(startTime, endTime);
    });

    // Update chart state when inputs change
    effect(() => {
      const mapData = this.mapDataValue;
      const metrics = this.extraMetricsValue;
      if (!mapData || mapData.time.length === 0 || !this.chart) {
        return;
      }

      // `metrics` dependency ensures we refresh datasets when requested metrics change
      void metrics;
      this.updateChart();
    });
  }

  public ngAfterViewInit(): void {
    setTimeout(() => {
      this.initChart();
    }, 100);
  }

  public ngOnDestroy(): void {
    if (this.chart) {
      this.chart.destroy();
    }
  }

  public toggleViewMode(): void {
    this.viewMode = this.viewMode === 'time' ? 'distance' : 'time';
    this.updateChart();
  }

  public zoomToRange(startTime: number, endTime: number): void {
    const mapData = this.mapDataValue;
    if (!this.chart || !mapData) {
      return;
    }

    let min = startTime;
    let max = endTime;

    if (this.viewMode === 'distance') {
      // Convert time to distance
      const startIndex = this.timeLabels.indexOf(startTime);
      const endIndex = this.timeLabels.indexOf(endTime);
      if (startIndex >= 0 && endIndex >= 0) {
        min = mapData.distance[startIndex];
        max = mapData.distance[endIndex];
      }
    }

    this.chart.zoomScale('x', { min, max });
  }

  public resetZoom(): void {
    if (!this.chart) {
      return;
    }

    this.chart.resetZoom();
  }

  /**
   * Handle chart zoom/pan events and translate them to interval selections
   */
  private onChartZoom(chart: Chart): void {
    const mapData = this.mapDataValue;
    if (!mapData) {
      return;
    }

    // Get the current visible range from the chart
    const xScale = chart.scales['x'];
    if (!xScale) {
      return;
    }

    const visibleMin = xScale.min;
    const visibleMax = xScale.max;

    // Check if we're at full zoom (original bounds)
    const originalMin =
      this.viewMode === 'time' ? new Date(mapData.time[0]).valueOf() : mapData.distance[0];
    const originalMax =
      this.viewMode === 'time'
        ? new Date(mapData.time[mapData.time.length - 1]).valueOf()
        : mapData.distance[mapData.distance.length - 1];

    // If we're at full zoom, clear the selection
    if (Math.abs(visibleMin - originalMin) < 1 && Math.abs(visibleMax - originalMax) < 1) {
      this.isUpdatingFromZoom = true;
      this.coordinatorService.clearSelection();
      this.isUpdatingFromZoom = false;
      return;
    }

    // Find the indices corresponding to the visible range
    let startIndex = 0;
    let endIndex = mapData.time.length - 1;

    if (this.viewMode === 'time') {
      // Find indices based on time
      for (let i = 0; i < mapData.time.length; i++) {
        const time = new Date(mapData.time[i]).valueOf();
        if (time >= visibleMin) {
          startIndex = i;
          break;
        }
      }
      for (let i = mapData.time.length - 1; i >= 0; i--) {
        const time = new Date(mapData.time[i]).valueOf();
        if (time <= visibleMax) {
          endIndex = i;
          break;
        }
      }
    } else {
      // Find indices based on distance
      for (let i = 0; i < mapData.distance.length; i++) {
        if (mapData.distance[i] >= visibleMin) {
          startIndex = i;
          break;
        }
      }
      for (let i = mapData.distance.length - 1; i >= 0; i--) {
        if (mapData.distance[i] <= visibleMax) {
          endIndex = i;
          break;
        }
      }
    }

    // Update the coordinator service with the new selection
    this.isUpdatingFromZoom = true;
    this.coordinatorService.selectInterval(startIndex, endIndex);
    this.isUpdatingFromZoom = false;
  }

  private initChart(): void {
    const canvasRef = this.chartCanvas();
    const mapData = this.mapDataValue;
    if (!canvasRef || !mapData || mapData.time.length === 0) {
      return;
    }

    const ctx = canvasRef.nativeElement.getContext('2d');
    if (!ctx) {
      return;
    }

    const config: ChartConfiguration = {
      type: 'line',
      data: {
        labels: this.getLabels(),
        datasets: this.getDatasets(),
      },
      options: this.getChartOptions(),
    };

    this.chart = new Chart(ctx, config);
  }

  private updateChart(): void {
    const mapData = this.mapDataValue;
    if (!this.chart || !mapData || mapData.time.length === 0) {
      this.initChart();
      return;
    }

    this.chart.data.labels = this.getLabels();
    this.chart.data.datasets = this.getDatasets();
    this.chart.options = this.getChartOptions();
    this.chart.update();
  }

  private getLabels(): (number | Date)[] {
    const mapData = this.mapDataValue;
    if (!mapData) {
      return [];
    }

    this.timeLabels = mapData.time.map((timestamp: string) => new Date(timestamp).valueOf());

    if (this.viewMode === 'time') {
      return this.timeLabels;
    } else {
      return mapData.distance;
    }
  }

  private getDatasets(): ChartDataset[] {
    const mapData = this.mapDataValue;
    if (!mapData) {
      return [];
    }

    const metrics = this.extraMetricsValue;
    const metricSettings = this.getMetricSettings();
    const datasets: ChartDataset[] = [];

    // Add speed dataset (convert to preferred unit)
    if (mapData.speed) {
      const speedData = mapData.speed.map((val) => this.convertSpeed(val));
      datasets.push({
        type: 'line',
        label: 'Speed',
        data: speedData,
        yAxisID: 'speed',
        spanGaps: true,
        hidden: false,
      });
    }

    // Add extra metrics
    if (mapData.extra_metrics) {
      for (const metric of metrics) {
        if (metric === 'speed') {
          continue;
        } // Already handled

        if (mapData.extra_metrics[metric]) {
          const settings = metricSettings[metric];
          datasets.push({
            type: 'line',
            label: this.getMetricLabel(metric),
            data: mapData.extra_metrics[metric] as number[],
            yAxisID: metric,
            spanGaps: true,
            hidden: settings?.hiddenByDefault || false,
            ...(metric === 'heart-rate'
              ? {
                  segment: {
                    borderColor: (ctx): string =>
                      this.zoneColor(
                        mapData,
                        'hr-zone',
                        ctx.p0DataIndex,
                        HR_ZONE_COLORS,
                        DEFAULT_HEART_RATE_COLOR,
                      ),
                  },
                }
              : {}),
            ...(metric === 'power'
              ? {
                  segment: {
                    borderColor: (ctx): string =>
                      this.zoneColor(
                        mapData,
                        'zone',
                        ctx.p0DataIndex,
                        FTP_ZONE_COLORS,
                        DEFAULT_POWER_COLOR,
                      ),
                  },
                }
              : {}),
          });
        }
      }
    }

    // Add elevation dataset with area fill
    if (mapData.elevation) {
      datasets.push({
        type: 'line',
        label: 'Elevation',
        data: mapData.elevation,
        yAxisID: 'elevation',
        fill: 'start',
        spanGaps: true,
        hidden: false,
      });
    }

    return datasets;
  }

  private getMetricLabel(metric: string): string {
    const labels: Record<string, string> = {
      speed: 'Speed',
      elevation: 'Elevation',
      'heart-rate': 'Heart Rate',
      cadence: 'Cadence',
      temperature: 'Temperature',
      power: 'Power',
    };
    return labels[metric] || metric;
  }

  private getChartOptions(): ChartOptions {
    const mapData = this.mapDataValue;
    if (!mapData) {
      return {};
    }

    const metricSettings = this.getMetricSettings();

    return {
      maintainAspectRatio: false,
      animation: false,
      scales: {
        x: {
          type: this.viewMode === 'time' ? 'time' : 'linear',
          time: this.viewMode === 'time' ? { unit: 'minute' } : undefined,
          min: this.viewMode === 'time' ? new Date(mapData.time[0]).valueOf() : mapData.distance[0],
          max:
            this.viewMode === 'time'
              ? new Date(mapData.time[mapData.time.length - 1]).valueOf()
              : mapData.distance[mapData.distance.length - 1],
          ticks: {
            callback: (val: string | number): string => {
              if (this.viewMode === 'distance') {
                const numVal = val as number;
                return `${numVal % 1 ? numVal.toFixed(1) : numVal} km`;
              }
              return new Date(val as number).toTimeString().substr(0, 5);
            },
          },
        },
        ...this.buildYAxes(metricSettings),
      },
      elements: {
        point: {
          radius: 0,
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
          display: true,
          onClick: (
            e: unknown,
            legendItem: { datasetIndex?: number },
            legend: { chart: Chart },
          ): void => {
            const chart = legend.chart;
            const index = legendItem.datasetIndex!;
            const meta = chart.getDatasetMeta(index);
            const isHidden = meta.hidden === null ? false : meta.hidden;
            meta.hidden = !isHidden;
            const yAxisID = meta.yAxisID;
            if (yAxisID && chart.options.scales![yAxisID]) {
              (chart.options.scales![yAxisID] as { display?: boolean }).display = !meta.hidden;
            }
            chart.update();
          },
        },
        tooltip: {
          callbacks: {
            title: (tooltipItems: TooltipItem<'line'>[]): string => {
              const items = tooltipItems;
              if (!items[0]) {
                return '';
              }
              const x = items[0].parsed.x as number;
              if (this.viewMode === 'distance') {
                return `${x.toFixed(2)} km`;
              }
              return new Date(x).toTimeString().substr(0, 5);
            },
            label: (tooltipItem: unknown): string => {
              type TooltipItem = {
                dataset?: { label?: string; yAxisID?: string };
                formattedValue?: string;
                raw?: unknown;
              };
              const ti = tooltipItem as TooltipItem;
              const yAxisID = ti.dataset?.yAxisID as string | undefined;
              const settings = yAxisID ? metricSettings[yAxisID] : undefined;
              let value = ti.formattedValue ?? '';
              if (settings && settings.formatter) {
                value = settings.formatter(ti.raw as number);
              }
              return `${ti.dataset?.label}: ${value}`;
            },
          },
        },
        zoom: {
          limits: {
            x: { min: 'original', max: 'original' },
            y: { min: 'original', max: 'original' },
          },
          zoom: {
            drag: {
              enabled: true,
            },
            wheel: {
              enabled: true,
            },
            mode: 'x',
            onZoomComplete: ({ chart }: { chart: Chart }): void => {
              this.onChartZoom(chart);
            },
          },
        },
      },
    };
  }

  private buildYAxes(metricSettings: Record<string, MetricConfig>): Record<string, unknown> {
    const axes: Record<string, unknown> = {};

    for (const metric of Object.keys(metricSettings)) {
      if (metricSettings[metric].yaxis === false) {
        continue;
      }

      const yaxisConfig = metricSettings[metric].yaxis;
      const isYaxisObject = typeof yaxisConfig === 'object';

      axes[metric] = {
        display: !metricSettings[metric].hiddenByDefault,
        position: isYaxisObject && yaxisConfig.position ? yaxisConfig.position : 'left',
        ...(isYaxisObject ? yaxisConfig : {}),
        ticks: {
          callback: (val: number): string | number => {
            const settings = metricSettings[metric];
            if (settings.formatterYaxis && settings.labelFormatter) {
              return settings.labelFormatter(val);
            }
            return val;
          },
        },
      };
    }

    return axes;
  }

  private getMetricSettings(): Record<string, MetricConfig> {
    const speedUnit = this.speedUnit();

    return {
      speed: {
        formatter: (val: number) => `${val?.toFixed(2) ?? '-'} ${speedUnit === 'mph' ? 'mph' : 'km/h'}`,
        labelFormatter: (val: number) => `${val} ${speedUnit === 'mph' ? 'mph' : 'km/h'}`,
        formatterYaxis: true,
        yaxis: { min: 0 },
      },
      elevation: {
        formatter: (val: number) => `${val !== null ? val.toFixed(2) : '-'} m`,
        labelFormatter: (val: number) => `${val} m`,
        formatterYaxis: true,
        yaxis: { position: 'right' },
      },
      'heart-rate': {
        formatter: (val: number) => `${val ?? '-'} bpm`,
        labelFormatter: (val: number) => `${val} bpm`,
        formatterYaxis: true,
        hiddenByDefault: true,
        yaxis: {},
      },
      cadence: {
        formatter: (val: number) => `${val ?? '-'}`,
        labelFormatter: (val: number) => `${val}`,
        formatterYaxis: true,
        hiddenByDefault: true,
        yaxis: { min: 0 },
      },
      temperature: {
        formatter: (val: number) => `${val ?? '-'} °C`,
        labelFormatter: (val: number) => `${val} °C`,
        formatterYaxis: true,
        hiddenByDefault: true,
        yaxis: {},
      },
      power: {
        formatter: (val: number) =>
          `${val !== null && val !== undefined ? val.toFixed(0) : '-'} W`,
        labelFormatter: (val: number) => `${(val ?? 0).toFixed(0)} W`,
        formatterYaxis: true,
        hiddenByDefault: true,
        yaxis: { min: 0, position: 'right' },
      },
    };
  }

  private zoneColor(
    mapData: MapDataDetails,
    metricKey: string,
    index: number | undefined,
    palette: Record<number, string>,
    fallback: string,
  ): string {
    if (!mapData?.extra_metrics || index === undefined) {
      return fallback;
    }

    const zones = mapData.extra_metrics[metricKey] as (number | null | undefined)[] | undefined;
    if (!zones || index < 0 || index >= zones.length) {
      return fallback;
    }

    const zone = zones[index];
    if (typeof zone !== 'number') {
      return fallback;
    }

    return palette[zone] ?? fallback;
  }

  private convertSpeed(value: number | null | undefined): number | null {
    if (value === null || value === undefined) {
      return null;
    }

    const unit = this.speedUnit();

    if (unit === 'mph') {
      return value * 2.23694;
    }

    // Default to km/h when unit is not mph
    return value * 3.6;
  }
}
