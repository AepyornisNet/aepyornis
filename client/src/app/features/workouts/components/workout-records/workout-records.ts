import { ChangeDetectionStrategy, Component, inject, input } from '@angular/core';

import { TranslatePipe } from '@ngx-translate/core';
import { WorkoutIntervalRecord } from '../../../../core/types/workout';
import { WorkoutDetailCoordinatorService } from '../../services/workout-detail-coordinator.service';
import { FormatSpeedPipe } from '../../../../core/pipes/format-speed.pipe';
import { FormatDurationPipe } from '../../../../core/pipes/format-duration.pipe';
import { FormatDistancePipe } from '../../../../core/pipes/format-distance.pipe';

@Component({
  selector: 'app-workout-records',
  imports: [TranslatePipe, FormatDistancePipe, FormatDurationPipe, FormatSpeedPipe],
  templateUrl: './workout-records.html',
  styleUrl: './workout-records.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class WorkoutRecordsComponent {
  private readonly coordinatorService = inject(WorkoutDetailCoordinatorService);
  public readonly records = input.required<WorkoutIntervalRecord[]>();
  public readonly workoutType = input.required<string>();

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
