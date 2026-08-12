import { formatNumber } from '@angular/common';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  effect,
  inject,
  input,
  LOCALE_ID,
  signal,
} from '@angular/core';
import { firstValueFrom } from 'rxjs';
import { _, TranslatePipe } from '@ngx-translate/core';
import { getMetricDef } from '../../../../core/config/metrics';

import { Api } from '../../../../core/services/api';
import {
  WorkoutDetail,
  WorkoutRangeStats,
} from '../../../../core/types/workout';
import {
  IntervalSelection,
  WorkoutDetailCoordinatorService,
} from '../../services/workout-detail-coordinator.service';
import { FormatDurationPipe } from '../../../../core/pipes/format-duration.pipe';
import { FormatDistancePipe } from '../../../../core/pipes/format-distance.pipe';
import { FormatElevationPipe } from '../../../../core/pipes/format-elevation.pipe';
import { FormatSpeedPipe } from '../../../../core/pipes/format-speed.pipe';

type NumericRangeStatKey = {
  [K in keyof WorkoutRangeStats]: WorkoutRangeStats[K] extends number | undefined ? K : never;
}[keyof WorkoutRangeStats];

type RangeStatConfig = {
  key: string;
  labelKey: string;
  unit: string;
  decimals?: number;
  averageField?: NumericRangeStatKey;
  movingField?: NumericRangeStatKey;
  minField?: NumericRangeStatKey;
  maxField?: NumericRangeStatKey;
  metricKey?: string;
  ignoreZero?: boolean;
};

type WorkoutStatRow = {
  labelKey: string;
  value?: string;
};

type WorkoutStatCard = {
  key: string;
  labelKey: string;
  rows: WorkoutStatRow[];
};

const RANGE_CONFIGS: RangeStatConfig[] = [
  {
    key: 'cadence',
    labelKey: _('Cadence'),
    unit: 'rpm',
    decimals: 0,
    averageField: 'average_cadence',
    minField: 'min_cadence',
    maxField: 'max_cadence',
    metricKey: 'cadence',
  },
  {
    key: 'heart-rate',
    labelKey: _('Heart rate'),
    unit: 'bpm',
    decimals: 0,
    averageField: 'average_heart_rate',
    minField: 'min_heart_rate',
    maxField: 'max_heart_rate',
    metricKey: 'heart-rate',
  },
  {
    key: 'respiration-rate',
    labelKey: _('Respiration rate'),
    unit: 'bpm',
    decimals: 0,
    averageField: 'average_respiration_rate',
    minField: 'min_respiration_rate',
    maxField: 'max_respiration_rate',
    metricKey: 'heart-rate',
  },
  {
    key: 'power',
    labelKey: _('Power'),
    unit: 'W',
    decimals: 0,
    averageField: 'average_power',
    minField: 'min_power',
    maxField: 'max_power',
    metricKey: 'power',
  },
  {
    key: 'slope',
    labelKey: _('Slope'),
    unit: '%',
    decimals: 1,
    averageField: 'average_slope',
    minField: 'min_slope',
    maxField: 'max_slope',
    ignoreZero: false,
  },
  {
    key: 'temperature',
    labelKey: _('Temperature'),
    unit: '°C',
    decimals: 1,
    ignoreZero: false,
    averageField: 'average_temperature',
    minField: 'min_temperature',
    maxField: 'max_temperature',
    metricKey: 'temperature',
  },
];

import { AppIcon } from '../../../../core/components/app-icon/app-icon';

@Component({
  selector: 'app-workout-statistics',
  imports: [TranslatePipe, AppIcon],
  templateUrl: './workout-statistics.html',
  styleUrl: './workout-statistics.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class WorkoutStatisticsComponent {
  public readonly workout = input<WorkoutDetail | null>(null);
  private readonly locale = inject(LOCALE_ID);
  private readonly api = inject(Api);
  private readonly coordinator = inject(WorkoutDetailCoordinatorService);

  private readonly stats = signal<WorkoutRangeStats | null>(null);
  private readonly loading = signal(false);
  private formatDurationPipe = inject(FormatDurationPipe);
  private formatDistancePipe = inject(FormatDistancePipe);
  private formatElevationPipe = inject(FormatElevationPipe);
  private formatSpeedPipe = inject(FormatSpeedPipe);
  private requestId = 0;

  public constructor() {
    effect(() => {
      const workout = this.workout();
      const selection = this.coordinator.selectedInterval();
      void this.loadStats(workout, selection);
    });
  }

  public readonly cards = computed<WorkoutStatCard[]>(() => {
    const workout = this.workout();
    const stats = this.stats();
    if (!workout || !stats) {
      return [];
    }

    const selection = this.coordinator.selectedInterval();
    const availableMetrics = workout.records?.extra_metrics ?? [];

    const cards: WorkoutStatCard[] = [];

    const distanceCard = this.buildDistanceCard(stats, selection, workout.laps?.length ?? 0);
    if (distanceCard) {
      cards.push(distanceCard);
    }

    const speedCard = this.buildSpeedCard(stats, workout.type);
    if (speedCard) {
      cards.push(speedCard);
    }

    const elevationCard = this.buildElevationCard(stats);
    if (elevationCard) {
      cards.push(elevationCard);
    }

    RANGE_CONFIGS.forEach((config) => {
      const rangeCard = this.buildRangeCard(stats, config, availableMetrics);
      if (rangeCard) {
        cards.push(rangeCard);
      }
    });

    return cards;
  });

  public getCardIconClass(key: string): string {
    return getMetricDef(key).colorClass;
  }

  public getCardIcon(key: string): string {
    return getMetricDef(key).icon;
  }

  public hasStatistics(): boolean {
    return this.cards().length > 0;
  }

  private async loadStats(
    workout: WorkoutDetail | null,
    selection: IntervalSelection | null,
  ): Promise<void> {
    if (!workout?.id || !workout.records?.details?.distance?.length) {
      this.stats.set(null);
      this.loading.set(false);
      return;
    }

    const params =
      selection && selection.startIndex >= 0 && selection.endIndex >= selection.startIndex
        ? { start_index: selection.startIndex, end_index: selection.endIndex }
        : undefined;

    const currentRequest = ++this.requestId;
    this.loading.set(true);

    try {
      const response = await firstValueFrom(this.api.getWorkoutRangeStats(workout.id, params));
      if (this.requestId !== currentRequest) {
        return;
      }
      this.stats.set(response.results);
    } catch (error) {
      if (this.requestId === currentRequest) {
        console.error('Failed to load workout range stats', error);
        this.stats.set(null);
      }
    } finally {
      if (this.requestId === currentRequest) {
        this.loading.set(false);
      }
    }
  }

  private buildDistanceCard(
    stats: WorkoutRangeStats,
    selection: IntervalSelection | null,
    lapCount: number,
  ): WorkoutStatCard | null {
    const selectionActive = Boolean(selection);
    const rows: WorkoutStatRow[] = [];

    if (stats.distance > 0) {
      rows.push({
        labelKey: selectionActive ? _('Distance') : _('Total distance'),
        value: this.formatDistancePipe.transform(stats.distance),
      });
    }

    if (stats.duration > 0) {
      rows.push({
        labelKey: _('Duration'),
        value: this.formatDurationPipe.transform(stats.duration),
      });
    }

    const pauseDuration = stats.pause_duration;
    const noPauseDuration = stats.moving_duration;

    if (
      typeof noPauseDuration === 'number' &&
      Number.isFinite(noPauseDuration) &&
      noPauseDuration >= 0
    ) {
      rows.push({
        labelKey: _('Duration (No pause)'),
        value: this.formatDurationPipe.transform(noPauseDuration),
      });
    }

    if (typeof pauseDuration === 'number' && Number.isFinite(pauseDuration) && pauseDuration >= 0) {
      rows.push({
        labelKey: _('Time paused'),
        value: this.formatDurationPipe.transform(pauseDuration),
      });
    }

    if (!selectionActive && lapCount > 0) {
      rows.push({ labelKey: _('Laps'), value: lapCount.toString() });
    }

    return rows.length
      ? {
        key: 'distance-summary',
        labelKey: _('Distance'),
        rows,
      }
      : null;
  }

  private selectionBoundsMs(
    workout: WorkoutDetail,
    selection: IntervalSelection | null,
  ): { startMs: number; endMs: number } | null {
    if (!selection) {
      return null;
    }

    const times = workout.records?.details?.time;
    if (!times || times.length === 0) {
      return null;
    }

    const maxIdx = times.length - 1;
    const startIdx = Math.max(0, Math.min(selection.startIndex, maxIdx));
    const endIdx = Math.max(startIdx, Math.min(selection.endIndex, maxIdx));

    const startMs = new Date(times[startIdx]).getTime();
    const endMs = new Date(times[endIdx]).getTime();
    if (!Number.isFinite(startMs) || !Number.isFinite(endMs) || endMs < startMs) {
      return null;
    }

    return { startMs, endMs };
  }

  private buildSpeedCard(stats: WorkoutRangeStats, workoutType: string): WorkoutStatCard | null {
    const rows: WorkoutStatRow[] = [];

    if (stats.average_speed) {
      rows.push({
        labelKey: _('Average'),
        value: this.formatSpeedPipe.transform(stats.average_speed, workoutType),
      });
    }

    if (stats.average_speed_no_pause) {
      rows.push({
        labelKey: _('Average (no pause)'),
        value: this.formatSpeedPipe.transform(stats.average_speed_no_pause, workoutType),
      });
    }

    if (stats.min_speed) {
      rows.push({
        labelKey: _('Minimum'),
        value: this.formatSpeedPipe.transform(stats.min_speed, workoutType),
      });
    }

    if (stats.max_speed) {
      rows.push({
        labelKey: _('Maximum'),
        value: this.formatSpeedPipe.transform(stats.max_speed),
      });
    }

    return rows.length
      ? {
        key: 'speed',
        labelKey: _('Speed'),
        rows,
      }
      : null;
  }

  private buildElevationCard(stats: WorkoutRangeStats): WorkoutStatCard | null {
    const rows: WorkoutStatRow[] = [];

    if (stats.total_up > 0) {
      rows.push({
        labelKey: _('Total up'),
        value: this.formatElevationPipe.transform(stats.total_up),
      });
    }

    if (stats.total_down > 0) {
      rows.push({
        labelKey: _('Total down'),
        value: this.formatElevationPipe.transform(stats.total_down),
      });
    }

    if (stats.max_elevation > stats.min_elevation) {
      const elevationRange = `${this.formatElevationPipe.transform(stats.min_elevation)} - ${this.formatElevationPipe.transform(
        stats.max_elevation,
      )}`;

      rows.push({
        labelKey: _('Elevation range'),
        value: elevationRange,
      });
    }

    return rows.length
      ? {
        key: 'elevation-summary',
        labelKey: _('Elevation'),
        rows,
      }
      : null;
  }

  private buildRangeCard(
    stats: WorkoutRangeStats,
    config: RangeStatConfig,
    availableMetrics: string[],
  ): WorkoutStatCard | null {
    if (config.metricKey && !availableMetrics.includes(config.metricKey)) {
      return null;
    }

    const rows: WorkoutStatRow[] = [];
    const average = this.resolveValue(stats, config.averageField, config.ignoreZero !== false);
    const moving = this.resolveValue(stats, config.movingField, config.ignoreZero !== false);
    const min = this.resolveValue(stats, config.minField, config.ignoreZero !== false);
    const max = this.resolveValue(stats, config.maxField, config.ignoreZero !== false);

    if (average !== undefined) {
      rows.push({
        labelKey: _('Average'),
        value: this.formatRangeValue(average, config.unit, config.decimals),
      });
    }

    if (moving !== undefined && config.movingField) {
      rows.push({
        labelKey: _('Average (no pause)'),
        value: this.formatRangeValue(moving, config.unit, config.decimals),
      });
    }

    if (min !== undefined) {
      rows.push({
        labelKey: _('Minimum'),
        value: this.formatRangeValue(min, config.unit, config.decimals),
      });
    }

    if (max !== undefined) {
      rows.push({
        labelKey: _('Maximum'),
        value: this.formatRangeValue(max, config.unit, config.decimals),
      });
    }

    return rows.length
      ? {
        key: config.key,
        labelKey: config.labelKey,
        rows,
      }
      : null;
  }

  private resolveValue(
    stats: WorkoutRangeStats,
    field: NumericRangeStatKey | undefined,
    ignoreZero: boolean,
  ): number | undefined {
    if (!field) {
      return undefined;
    }

    const value = stats[field];
    if (typeof value !== 'number' || Number.isNaN(value)) {
      return undefined;
    }

    if (ignoreZero && value === 0) {
      return undefined;
    }

    return value;
  }

  private formatRangeValue(value: number, unit: string, decimals: number | undefined): string {
    if (value === undefined || Number.isNaN(value)) {
      return '-';
    }

    if (unit === `%`) {
      value = value * 100;
    }
    const digits = decimals !== undefined && decimals > 0 ? `1.${decimals}-${decimals}` : '1.0-0';
    const formatted = formatNumber(value, this.locale, digits);
    return unit ? `${formatted} ${unit}` : formatted;
  }
}
