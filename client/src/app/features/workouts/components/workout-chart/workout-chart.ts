import {
  Component,
  Input,
  OnChanges,
  SimpleChanges,
  ElementRef,
  ViewChild,
  AfterViewInit,
  OnDestroy,
  inject,
  effect
} from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  Chart,
  ChartConfiguration,
  ChartOptions,
  ChartDataset,
  CategoryScale,
  LinearScale,
  TimeScale,
  PointElement,
  LineController,
  LineElement,
  Filler,
  Legend,
  Tooltip,
  Decimation,
  Colors
} from 'chart.js';
import 'chartjs-adapter-date-fns';
import zoomPlugin from 'chartjs-plugin-zoom';
import { MapDataDetails } from '../../../../core/types/workout';
import { WorkoutDetailCoordinatorService } from '../../services/workout-detail-coordinator.service';

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
  zoomPlugin
);

interface MetricConfig {
  formatter?: (val: number) => string;
  labelFormatter?: (val: number) => string;
  formatterYaxis?: boolean;
  yaxis?: boolean | { min?: number; max?: number; position?: string };
  hiddenByDefault?: boolean;
}

@Component({
  selector: 'app-workout-chart',
  imports: [CommonModule],
  templateUrl: './workout-chart.html',
  styleUrl: './workout-chart.scss'
})
export class WorkoutChartComponent implements AfterViewInit, OnChanges, OnDestroy {
  @ViewChild('chartCanvas', { static: false }) chartCanvas!: ElementRef<HTMLCanvasElement>;

  @Input() mapData?: MapDataDetails;
  @Input() extraMetrics: string[] = [];

  private coordinatorService = inject(WorkoutDetailCoordinatorService);
  private chart?: Chart;
  private timeLabels: number[] = [];
  private isUpdatingFromZoom = false; // Flag to prevent infinite loops
  viewMode: 'time' | 'distance' = 'time';

  constructor() {
    // React to interval selection changes from the coordinator service
    effect(() => {
      const selection = this.coordinatorService.selectedInterval();

      // Don't react if the change came from our own zoom
      if (this.isUpdatingFromZoom) {
        return;
      }

      if (!this.mapData || !selection) {
        // Only reset zoom if there's no selection
        if (!selection && this.chart) {
          this.resetZoom();
        }
        return;
      }

      // Zoom to the selected interval
      const startTime = new Date(this.mapData.time[selection.startIndex]).getTime();
      const endTime = new Date(this.mapData.time[selection.endIndex]).getTime();
      this.zoomToRange(startTime, endTime);
    });
  }

  ngAfterViewInit() {
    setTimeout(() => {
      this.initChart();
    }, 100);
  }

  ngOnChanges(changes: SimpleChanges) {
    if (changes['mapData'] && !changes['mapData'].firstChange) {
      this.updateChart();
    }
  }

  ngOnDestroy() {
    if (this.chart) {
      this.chart.destroy();
    }
  }

  toggleViewMode() {
    this.viewMode = this.viewMode === 'time' ? 'distance' : 'time';
    this.updateChart();
  }

  zoomToRange(startTime: number, endTime: number) {
    if (!this.chart || !this.mapData) {
      return;
    }

    let min = startTime;
    let max = endTime;

    if (this.viewMode === 'distance') {
      // Convert time to distance
      const startIndex = this.timeLabels.indexOf(startTime);
      const endIndex = this.timeLabels.indexOf(endTime);
      if (startIndex >= 0 && endIndex >= 0) {
        min = this.mapData.distance[startIndex];
        max = this.mapData.distance[endIndex];
      }
    }

    this.chart.zoomScale('x', { min, max });
  }

  resetZoom() {
    if (!this.chart) {
      return;
    }

    this.chart.resetZoom();
  }

  /**
   * Handle chart zoom/pan events and translate them to interval selections
   */
  private onChartZoom(chart: Chart) {
    if (!this.mapData) {
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
    const originalMin = this.viewMode === 'time'
      ? new Date(this.mapData.time[0]).valueOf()
      : this.mapData.distance[0];
    const originalMax = this.viewMode === 'time'
      ? new Date(this.mapData.time[this.mapData.time.length - 1]).valueOf()
      : this.mapData.distance[this.mapData.distance.length - 1];

    // If we're at full zoom, clear the selection
    if (Math.abs(visibleMin - originalMin) < 1 && Math.abs(visibleMax - originalMax) < 1) {
      this.isUpdatingFromZoom = true;
      this.coordinatorService.clearSelection();
      this.isUpdatingFromZoom = false;
      return;
    }

    // Find the indices corresponding to the visible range
    let startIndex = 0;
    let endIndex = this.mapData.time.length - 1;

    if (this.viewMode === 'time') {
      // Find indices based on time
      for (let i = 0; i < this.mapData.time.length; i++) {
        const time = new Date(this.mapData.time[i]).valueOf();
        if (time >= visibleMin) {
          startIndex = i;
          break;
        }
      }
      for (let i = this.mapData.time.length - 1; i >= 0; i--) {
        const time = new Date(this.mapData.time[i]).valueOf();
        if (time <= visibleMax) {
          endIndex = i;
          break;
        }
      }
    } else {
      // Find indices based on distance
      for (let i = 0; i < this.mapData.distance.length; i++) {
        if (this.mapData.distance[i] >= visibleMin) {
          startIndex = i;
          break;
        }
      }
      for (let i = this.mapData.distance.length - 1; i >= 0; i--) {
        if (this.mapData.distance[i] <= visibleMax) {
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

  private initChart() {
    if (!this.chartCanvas || !this.mapData || this.mapData.time.length === 0) {
      return;
    }

    const ctx = this.chartCanvas.nativeElement.getContext('2d');
    if (!ctx) {
      return;
    }

    const config: ChartConfiguration = {
      type: 'line',
      data: {
        labels: this.getLabels(),
        datasets: this.getDatasets()
      },
      options: this.getChartOptions()
    };

    this.chart = new Chart(ctx, config);
  }

  private updateChart() {
    if (!this.chart) {
      this.initChart();
      return;
    }

    this.chart.data.labels = this.getLabels();
    this.chart.data.datasets = this.getDatasets();
    this.chart.options = this.getChartOptions();
    this.chart.update();
  }

  private getLabels(): (number | Date)[] {
    if (!this.mapData) {
      return [];
    }

    this.timeLabels = this.mapData.time.map(t => new Date(t).valueOf());

    if (this.viewMode === 'time') {
      return this.timeLabels;
    } else {
      return this.mapData.distance;
    }
  }

  private getDatasets(): ChartDataset[] {
    if (!this.mapData) {
      return [];
    }

    const metricSettings = this.getMetricSettings();
    const datasets: ChartDataset[] = [];

    // Add speed dataset
    if (this.mapData.speed) {
      datasets.push({
        type: 'line',
        label: 'Speed',
        data: this.mapData.speed,
        yAxisID: 'speed',
        spanGaps: true,
        hidden: false
      });
    }

    // Add elevation dataset with area fill
    if (this.mapData.elevation) {
      datasets.push({
        type: 'line',
        label: 'Elevation',
        data: this.mapData.elevation,
        yAxisID: 'elevation',
        fill: 'start',
        spanGaps: true,
        hidden: false
      });
    }

    // Add extra metrics
    if (this.mapData.extra_metrics) {
      for (const metric of this.extraMetrics) {
        if (metric === 'speed') continue; // Already handled

        if (this.mapData.extra_metrics[metric]) {
          const settings = metricSettings[metric];
          datasets.push({
            type: 'line',
            label: this.getMetricLabel(metric),
            data: this.mapData.extra_metrics[metric] as number[],
            yAxisID: metric,
            spanGaps: true,
            hidden: settings?.hiddenByDefault || false
          });
        }
      }
    }

    return datasets;
  }

  private getMetricLabel(metric: string): string {
    const labels: Record<string, string> = {
      'speed': 'Speed',
      'elevation': 'Elevation',
      'heart-rate': 'Heart Rate',
      'cadence': 'Cadence',
      'temperature': 'Temperature'
    };
    return labels[metric] || metric;
  }

  private getChartOptions(): ChartOptions {
    if (!this.mapData) {
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
          min: this.viewMode === 'time'
            ? new Date(this.mapData.time[0]).valueOf()
            : this.mapData.distance[0],
          max: this.viewMode === 'time'
            ? new Date(this.mapData.time[this.mapData.time.length - 1]).valueOf()
            : this.mapData.distance[this.mapData.distance.length - 1],
          ticks: {
            callback: (val: string | number) => {
              if (this.viewMode === 'distance') {
                const numVal = val as number;
                return `${numVal % 1 ? numVal.toFixed(1) : numVal} km`;
              }
              return new Date(val as number).toTimeString().substr(0, 5);
            }
          }
        },
        ...this.buildYAxes(metricSettings)
      },
      elements: {
        point: {
          radius: 0
        }
      },
      interaction: {
        mode: 'index',
        intersect: false
      },
      plugins: {
        decimation: {
          enabled: true,
          algorithm: 'lttb'
        },
        legend: {
          display: true,
          onClick: (e, legendItem, legend) => {
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
          }
        },
        tooltip: {
          callbacks: {
            title: (tooltipItems) => {
              if (!tooltipItems[0]) {
                return '';
              }
              const x = tooltipItems[0].parsed.x;
              if (this.viewMode === 'distance') {
                return `${x.toFixed(2)} km`;
              }
              return new Date(x).toTimeString().substr(0, 5);
            },
            label: (tooltipItem) => {
              const settings = metricSettings[tooltipItem.dataset.yAxisID as string];
              let value = tooltipItem.formattedValue;
              if (settings && settings.formatter) {
                value = settings.formatter(tooltipItem.raw as number);
              }
              return `${tooltipItem.dataset.label}: ${value}`;
            }
          }
        },
        zoom: {
          limits: {
            x: { min: 'original', max: 'original' },
            y: { min: 'original', max: 'original' }
          },
          zoom: {
            drag: {
              enabled: true
            },
            wheel: {
              enabled: true
            },
            mode: 'x',
            onZoomComplete: ({ chart }) => {
              this.onChartZoom(chart);
            }
          }
        }
      }
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
          callback: (val: number) => {
            const settings = metricSettings[metric];
            if (settings.formatterYaxis && settings.labelFormatter) {
              return settings.labelFormatter(val);
            }
            return val;
          }
        }
      };
    }

    return axes;
  }

  private getMetricSettings(): Record<string, MetricConfig> {
    return {
      speed: {
        formatter: (val: number) => `${val?.toFixed(2) ?? '-'} m/s`,
        labelFormatter: (val: number) => `${val} m/s`,
        formatterYaxis: true,
        yaxis: { min: 0 }
      },
      elevation: {
        formatter: (val: number) => `${val !== null ? val.toFixed(2) : '-'} m`,
        labelFormatter: (val: number) => `${val} m`,
        formatterYaxis: true,
        yaxis: { position: 'right' }
      },
      'heart-rate': {
        formatter: (val: number) => `${val ?? '-'} bpm`,
        labelFormatter: (val: number) => `${val} bpm`,
        formatterYaxis: true,
        hiddenByDefault: true,
        yaxis: {}
      },
      cadence: {
        formatter: (val: number) => `${val ?? '-'}`,
        labelFormatter: (val: number) => `${val}`,
        formatterYaxis: true,
        hiddenByDefault: true,
        yaxis: { min: 0 }
      },
      temperature: {
        formatter: (val: number) => `${val ?? '-'} °C`,
        labelFormatter: (val: number) => `${val} °C`,
        formatterYaxis: true,
        hiddenByDefault: true,
        yaxis: {}
      }
    };
  }
}
