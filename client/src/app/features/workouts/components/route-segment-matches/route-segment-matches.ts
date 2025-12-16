import { ChangeDetectionStrategy, Component, inject, input } from '@angular/core';

import { TranslatePipe } from '@ngx-translate/core';
import { RouteSegmentMatch } from '../../../../core/types/workout';
import { WorkoutDetailCoordinatorService } from '../../services/workout-detail-coordinator.service';
import { RouterLink } from '@angular/router';

@Component({
  selector: 'app-route-segment-matches',
  imports: [TranslatePipe, RouterLink],
  templateUrl: './route-segment-matches.html',
  styleUrl: './route-segment-matches.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class RouteSegmentMatchesComponent {
  private readonly coordinatorService = inject(WorkoutDetailCoordinatorService);
  public readonly matches = input.required<RouteSegmentMatch[]>();

  public formatDistance(meters: number): string {
    return (meters / 1000).toFixed(2);
  }

  public formatDuration(seconds?: number): string {
    if (!seconds || Number.isNaN(seconds)) {
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

  public selectMatch(match: RouteSegmentMatch): void {
    if (!this.hasIntervalIndexes(match)) {
      return;
    }

    if (this.isSelected(match)) {
      this.coordinatorService.clearSelection();
      return;
    }

    this.coordinatorService.selectInterval(match.start_index, match.end_index);
  }

  public isSelected(match: RouteSegmentMatch): boolean {
    if (!this.hasIntervalIndexes(match)) {
      return false;
    }

    return this.coordinatorService.isIntervalSelected(match.start_index, match.end_index);
  }

  private hasIntervalIndexes(match: RouteSegmentMatch): boolean {
    return (
      typeof match.start_index === 'number' &&
      typeof match.end_index === 'number' &&
      match.start_index >= 0 &&
      match.end_index >= match.start_index
    );
  }
}
