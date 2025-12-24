import { ChangeDetectionStrategy, Component, inject, input } from '@angular/core';

import { TranslatePipe } from '@ngx-translate/core';
import { WorkoutIntervalRecord } from '../../../../core/types/workout';
import { WorkoutDetailCoordinatorService } from '../../services/workout-detail-coordinator.service';

@Component({
  selector: 'app-workout-records',
  imports: [TranslatePipe],
  templateUrl: './workout-records.html',
  styleUrl: './workout-records.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class WorkoutRecordsComponent {
  private readonly coordinatorService = inject(WorkoutDetailCoordinatorService);
  public readonly records = input.required<WorkoutIntervalRecord[]>();

  public formatDistance(meters: number): string {
    return (meters / 1000).toFixed(2);
  }

  public formatDuration(seconds?: number): string {
    if (!seconds && seconds !== 0) {
      return '0:00';
    }

    const totalSeconds = Math.round(seconds);
    const hours = Math.floor(totalSeconds / 3600);
    const minutes = Math.floor((totalSeconds % 3600) / 60);
    const secs = totalSeconds % 60;

    if (hours > 0) {
      return `${hours}:${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
    }

    return `${minutes}:${secs.toString().padStart(2, '0')}`;
  }

  public formatSpeed(speed?: number): string {
    if (speed === undefined || speed === null || Number.isNaN(speed)) {
      return '-';
    }

    return `${(speed * 3.6).toFixed(2)} km/h`;
  }

  public selectRecord(record: WorkoutIntervalRecord): void {
    if (!this.hasIntervalIndexes(record)) {
      return;
    }

    if (this.isSelected(record)) {
      this.coordinatorService.clearSelection();
      return;
    }

    this.coordinatorService.selectInterval(record.start_index!, record.end_index!);
  }

  public isSelected(record: WorkoutIntervalRecord): boolean {
    if (!this.hasIntervalIndexes(record)) {
      return false;
    }

    return this.coordinatorService.isIntervalSelected(record.start_index!, record.end_index!);
  }

  private hasIntervalIndexes(record: WorkoutIntervalRecord): boolean {
    return (
      typeof record.start_index === 'number' &&
      typeof record.end_index === 'number' &&
      record.start_index >= 0 &&
      record.end_index >= record.start_index
    );
  }
}
