import {
  ChangeDetectionStrategy,
  Component,
  computed,
  effect,
  inject,
  input,
  signal,
} from '@angular/core';
import { TranslatePipe } from '@ngx-translate/core';

import { Api } from '../../../../core/services/api';
import { WorkoutBreakdown } from '../../../../core/types/workout';
import { User } from '../../../../core/services/user';
import { WorkoutDetailCoordinatorService } from '../../services/workout-detail-coordinator.service';
import { WorkoutDetailDataService } from '../../services/workout-detail-data.service';
import { FormatDistancePipe } from '../../../../core/pipes/format-distance.pipe';
import { FormatDurationPipe } from '../../../../core/pipes/format-duration.pipe';
import { FormatElevationPipe } from '../../../../core/pipes/format-elevation.pipe';
import { FormatSpeedPipe } from '../../../../core/pipes/format-speed.pipe';

@Component({
  selector: 'app-workout-breakdown',
  imports: [
    TranslatePipe,
    FormatDistancePipe,
    FormatDurationPipe,
    FormatElevationPipe,
    FormatSpeedPipe,
  ],
  templateUrl: './workout-breakdown.html',
  styleUrl: './workout-breakdown.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class WorkoutBreakdownComponent {
  private api = inject(Api);
  private dataService = inject(WorkoutDetailDataService);
  private userService = inject(User);
  private formatDistancePipe = inject(FormatDistancePipe);
  private coordinatorService = inject(WorkoutDetailCoordinatorService);

  public readonly workoutId = input<number | undefined>();
  public readonly totalDistance = input<number | undefined>();
  public readonly extraMetrics = input<string[]>([]);

  public readonly breakdown = signal<WorkoutBreakdown | null>(null);
  public readonly loading = signal(false);
  public readonly workoutHasLaps = signal(false);

  public readonly distanceUnit = computed<string>(() => {
    const info = this.userService.getUserInfo()();
    return info?.profile?.profile?.preferred_units?.distance ?? 'km';
  });

  public readonly speedUnit = computed<string>(() => {
    const info = this.userService.getUserInfo()();
    return info?.profile?.profile?.preferred_units?.speed ?? 'km/h';
  });

  public readonly elevationUnit = computed<string>(() => {
    const info = this.userService.getUserInfo()();
    return info?.profile?.profile?.preferred_units?.elevation ?? 'm';
  });

  public selectedIntervalIndex: number | null = null;
  public readonly breakdownMode = signal<'laps' | 'unit'>('laps');
  public readonly intervalDistance = signal<number>(1);

  public readonly availableIntervals = computed<number[]>(() => {
    const total = this.totalDistance();
    const totalConverted = this.formatDistancePipe.convert(total ?? 0);
    const options = [1, 2, 5, 10, 25];
    const intervals = options.filter((value) => value < Number(totalConverted));
    return intervals.length > 0 ? intervals : [1];
  });

  public readonly workout = this.dataService.workout();
  public constructor() {
    effect(() => {
      if (!this.workout || !this.workout.laps || this.workout.laps.length === 0) {
        this.workoutHasLaps.set(false);
        return;
      }

      this.workoutHasLaps.set(true);
    });

    effect(() => {
      const id = this.workoutId();

      if (!id) {
        this.breakdown.set(null);
        this.selectedIntervalIndex = null;
        return;
      }

      if (!this.workoutHasLaps) {
        this.intervalDistance.set(1);
      }

      this.loadBreakdown({ mode: 'auto' });
    });

    effect(() => {
      const selection = this.coordinatorService.selectedInterval();
      const items = this.breakdown()?.items;

      if (!selection || !items || items.length === 0) {
        this.selectedIntervalIndex = null;
        return;
      }

      const idx = items.findIndex(
        (item) =>
          item.start_index === selection.startIndex && item.end_index === selection.endIndex,
      );

      this.selectedIntervalIndex = idx >= 0 ? idx : null;
    });
  }

  private loadBreakdown(params: { mode?: 'auto' | 'laps' | 'unit'; count?: number }): void {
    const workoutId = this.workoutId();
    if (!workoutId) {
      return;
    }

    this.loading.set(true);
    const request$ = this.api.getWorkoutBreakdown(workoutId, {
      mode: params.mode,
      count: params.count,
    });

    request$.subscribe({
      next: (response) => {
        if (!response?.results) {
          this.breakdown.set(null);
          this.loading.set(false);
          return;
        }

        this.breakdown.set(response.results);
        const mode = (response.results.mode as 'laps' | 'unit') || 'unit';
        this.breakdownMode.set(mode);
        this.selectedIntervalIndex = null;
        this.loading.set(false);
      },
      error: () => {
        this.breakdown.set(null);
        this.loading.set(false);
      },
    });
  }

  public useLaps(): void {
    if (this.breakdownMode() === 'laps' && this.breakdown()) {
      this.selectedIntervalIndex = null;
      this.coordinatorService.clearSelection();
      return;
    }

    this.breakdownMode.set('laps');
    this.selectedIntervalIndex = null;
    this.coordinatorService.clearSelection();
    this.loadBreakdown({ mode: 'laps' });
  }

  public setIntervalDistance(distance: number): void {
    this.intervalDistance.set(distance);
    this.breakdownMode.set('unit');
    this.selectedIntervalIndex = null;
    this.coordinatorService.clearSelection();
    this.loadBreakdown({ mode: 'unit', count: distance });
  }

  public isIntervalActive(distance: number): boolean {
    return this.breakdownMode() === 'unit' && this.intervalDistance() === distance;
  }

  public selectItem(index: number): void {
    const items = this.breakdown()?.items;
    if (!items || !items[index]) {
      return;
    }

    if (this.selectedIntervalIndex === index) {
      this.selectedIntervalIndex = null;
      this.coordinatorService.clearSelection();
      return;
    }

    this.selectedIntervalIndex = index;
    const item = items[index];
    this.coordinatorService.selectInterval(item.start_index ?? -1, item.end_index ?? -1);
  }

  public hasExtraMetric(metric: string): boolean {
    return this.extraMetrics().includes(metric);
  }
}
