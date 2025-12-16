import { ChangeDetectionStrategy, Component, inject, input } from '@angular/core';

import { TranslatePipe } from '@ngx-translate/core';
import { ClimbSegment } from '../../../../core/types/workout';
import { WorkoutDetailCoordinatorService } from '../../services/workout-detail-coordinator.service';

@Component({
  selector: 'app-workout-climbs',
  imports: [TranslatePipe],
  templateUrl: './workout-climbs.html',
  styleUrl: './workout-climbs.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class WorkoutClimbsComponent {
  private readonly coordinatorService = inject(WorkoutDetailCoordinatorService);
  public readonly climbs = input.required<ClimbSegment[]>();

  public formatDistance(meters: number): string {
    return (meters / 1000).toFixed(2);
  }

  public formatElevation(meters: number): string {
    return meters.toFixed(0);
  }

  public selectClimb(climb: ClimbSegment): void {
    if (!this.hasIntervalIndexes(climb)) {
      return;
    }

    if (this.isSelected(climb)) {
      this.coordinatorService.clearSelection();
      return;
    }

    this.coordinatorService.selectInterval(climb.start_index, climb.end_index);
  }

  public isSelected(climb: ClimbSegment): boolean {
    if (!this.hasIntervalIndexes(climb)) {
      return false;
    }

    return this.coordinatorService.isIntervalSelected(climb.start_index, climb.end_index);
  }

  private hasIntervalIndexes(climb: ClimbSegment): boolean {
    return (
      typeof climb.start_index === 'number' &&
      typeof climb.end_index === 'number' &&
      climb.start_index >= 0 &&
      climb.end_index >= climb.start_index
    );
  }
}
