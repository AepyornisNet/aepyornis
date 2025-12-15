import { ChangeDetectionStrategy, Component, computed, inject, input } from '@angular/core';
import { TranslatePipe } from '@ngx-translate/core';
import { MapDataDetails } from '../../../../core/types/workout';
import {
  DEFAULT_HEART_RATE_COLOR,
  DEFAULT_POWER_COLOR,
  FTP_ZONE_COLORS,
  HR_ZONE_COLORS,
} from '../zone-colors';
import {
  IntervalSelection,
  WorkoutDetailCoordinatorService,
} from '../../services/workout-detail-coordinator.service';

const ZONE_ORDER: Record<'heart-rate' | 'power', number[]> = {
  'heart-rate': [1, 2, 3, 4, 5],
  power: [1, 2, 3, 4, 5, 6, 7],
};

type ZoneType = 'heart-rate' | 'power';
type ZoneSummary = {
  zone: number;
  seconds: number;
  percent: number;
  samples: number;
  color: string;
};

@Component({
  selector: 'app-workout-zone-distribution',
  imports: [TranslatePipe],
  templateUrl: './workout-zone-distribution.html',
  styleUrl: './workout-zone-distribution.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class WorkoutZoneDistributionComponent {
  public readonly mapData = input<MapDataDetails | undefined>();
  public readonly type = input<ZoneType>('heart-rate');
  private readonly coordinator = inject(WorkoutDetailCoordinatorService);

  public readonly distribution = computed<ZoneSummary[]>(() => {
    const details = this.mapData();
    const type = this.type();
    const selection = this.coordinator.selectedInterval();
    const metricKey = this.metricKey(type);

    if (!details || !details.extra_metrics || !details.extra_metrics[metricKey]) {
      return this.emptySummary(type);
    }

    const metricValues = details.extra_metrics[metricKey] as (number | null | undefined)[];
    if (!metricValues || metricValues.length === 0) {
      return this.emptySummary(type);
    }

    const { startIndex, endIndex } = this.resolveRange(selection, metricValues.length);
    const totals: Record<number, ZoneSummary> = {};
    const durations = details.duration || [];

    for (let i = startIndex; i <= endIndex; i++) {
      const zone = metricValues[i];
      if (!this.isZoneValue(zone)) {
        continue;
      }

      const currentDuration = durations[i] ?? durations[durations.length - 1] ?? 0;
      const prevDuration = i === 0 ? 0 : durations[i - 1] ?? currentDuration;
      const delta = Math.max(currentDuration - prevDuration, 0);

      if (!totals[zone]) {
        totals[zone] = {
          zone,
          seconds: 0,
          percent: 0,
          samples: 0,
          color: this.paletteFor(type)[zone] ?? this.defaultColor(type),
        };
      }

      totals[zone].seconds += delta;
      totals[zone].samples += 1;
    }

    const secondSum = Object.values(totals).reduce((acc, entry) => acc + entry.seconds, 0);
    const sampleSum = Object.values(totals).reduce((acc, entry) => acc + entry.samples, 0);
    const useSamplesForPercent = secondSum <= 0 && sampleSum > 0;
    const percentBase = useSamplesForPercent ? sampleSum : secondSum;

    return ZONE_ORDER[type].map((zone) => {
      const data = totals[zone];
      const seconds = data?.seconds ?? 0;
      const samples = data?.samples ?? 0;
      const basis = useSamplesForPercent ? samples : seconds;
      const percent = percentBase > 0 ? (basis / percentBase) * 100 : 0;

      return {
        zone,
        seconds,
        samples,
        percent,
        color: this.paletteFor(type)[zone] ?? this.defaultColor(type),
      };
    });
  });

  public readonly hasData = computed(() => this.distribution().some((entry) => entry.percent > 0));

  public formatDuration(seconds: number): string {
    if (!Number.isFinite(seconds) || seconds <= 0) {
      return '0s';
    }

    const rounded = Math.round(seconds);
    const hours = Math.floor(rounded / 3600);
    const minutes = Math.floor((rounded % 3600) / 60);
    const secs = rounded % 60;

    if (hours > 0) {
      return `${hours}h ${minutes}m ${secs}s`;
    }
    if (minutes > 0) {
      return `${minutes}m ${secs}s`;
    }
    return `${secs}s`;
  }

  public formatPercent(value: number): string {
    if (!Number.isFinite(value) || value <= 0) {
      return '0%';
    }

    return `${value.toFixed(1)}%`;
  }

  private emptySummary(type: ZoneType): ZoneSummary[] {
    return ZONE_ORDER[type].map((zone) => ({
      zone,
      seconds: 0,
      samples: 0,
      percent: 0,
      color: this.paletteFor(type)[zone] ?? this.defaultColor(type),
    }));
  }

  private metricKey(type: ZoneType): 'hr-zone' | 'zone' {
    return type === 'heart-rate' ? 'hr-zone' : 'zone';
  }

  private resolveRange(selection: IntervalSelection | null, length: number): {
    startIndex: number;
    endIndex: number;
  } {
    if (!selection || length === 0) {
      return { startIndex: 0, endIndex: Math.max(length - 1, 0) };
    }

    const start = Math.max(0, Math.min(selection.startIndex, length - 1));
    const end = Math.max(start, Math.min(selection.endIndex, length - 1));
    return { startIndex: start, endIndex: end };
  }

  private paletteFor(type: ZoneType): Record<number, string> {
    return type === 'heart-rate' ? HR_ZONE_COLORS : FTP_ZONE_COLORS;
  }

  private defaultColor(type: ZoneType): string {
    return type === 'heart-rate' ? DEFAULT_HEART_RATE_COLOR : DEFAULT_POWER_COLOR;
  }

  private isZoneValue(value: unknown): value is number {
    return typeof value === 'number' && Number.isFinite(value) && value > 0;
  }
}
