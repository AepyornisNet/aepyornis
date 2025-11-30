import { ChangeDetectionStrategy, Component, effect, inject, input } from '@angular/core';

import { MapDataDetails } from '../../../../core/types/workout';
import { WorkoutDetailCoordinatorService } from '../../services/workout-detail-coordinator.service';
import {
  IntervalData,
  WorkoutDetailIntervalService,
} from '../../services/workout-detail-interval.service';
import { TranslatePipe } from '@ngx-translate/core';

@Component({
  selector: 'app-workout-breakdown',
  imports: [TranslatePipe],
  templateUrl: './workout-breakdown.html',
  styleUrl: './workout-breakdown.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class WorkoutBreakdownComponent {
  public readonly mapData = input<MapDataDetails | undefined>();
  public readonly extraMetrics = input<string[]>([]);

  private coordinatorService = inject(WorkoutDetailCoordinatorService);
  private intervalService = inject(WorkoutDetailIntervalService);

  public intervalDistance = 1; // km
  public availableIntervals: number[] = [];
  public intervals: IntervalData[] = [];
  public selectedIntervalIndex: number | null = null;

  public constructor() {
    effect(() => {
      const data = this.mapData();
      const metrics = this.extraMetrics();
      if (!data) {
        this.availableIntervals = [];
        this.intervals = [];
        this.selectedIntervalIndex = null;
        return;
      }

      this.availableIntervals = this.intervalService.calculateAvailableIntervals(data);
      this.intervals = this.intervalService.calculateIntervals(
        data,
        this.intervalDistance,
        metrics,
      );

      if (
        this.selectedIntervalIndex !== null &&
        this.selectedIntervalIndex >= this.intervals.length
      ) {
        this.selectedIntervalIndex = null;
      }
    });
  }

  public setIntervalDistance(distance: number): void {
    this.intervalDistance = distance;
    this.selectedIntervalIndex = null;

    const data = this.mapData();
    if (data) {
      this.intervals = this.intervalService.calculateIntervals(
        data,
        this.intervalDistance,
        this.extraMetrics(),
      );
    }

    this.coordinatorService.clearSelection();
  }

  public selectInterval(index: number): void {
    if (this.selectedIntervalIndex === index) {
      // Deselect
      this.selectedIntervalIndex = null;
      this.coordinatorService.clearSelection();
    } else {
      // Select new interval
      this.selectedIntervalIndex = index;
      const interval = this.intervals[index];
      this.coordinatorService.selectInterval(interval.startIndex, interval.endIndex);
    }
  }

  public formatDuration(milliseconds: number): string {
    const totalSeconds = Math.floor(milliseconds / 1000);
    const hours = Math.floor(totalSeconds / 3600);
    const minutes = Math.floor((totalSeconds % 3600) / 60);
    const seconds = totalSeconds % 60;

    if (hours > 0) {
      return `${hours}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
    }
    return `${minutes}:${seconds.toString().padStart(2, '0')}`;
  }

  public formatSpeed(speedMps: number): string {
    return (speedMps * 3.6).toFixed(2); // Convert m/s to km/h
  }

  public hasExtraMetric(metric: string): boolean {
    return this.extraMetrics().includes(metric);
  }
}
