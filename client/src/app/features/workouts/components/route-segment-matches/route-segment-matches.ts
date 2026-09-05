import { ChangeDetectionStrategy, Component, inject, input } from '@angular/core';

import { TranslatePipe } from '@ngx-translate/core';
import { RouteSegmentMatch } from '../../../../core/types/workout';
import { WorkoutDetailCoordinatorService } from '../../services/workout-detail-coordinator.service';
import { RouterLink } from '@angular/router';
import { FormatDistancePipe } from '../../../../core/pipes/format-distance.pipe';
import { FormatDurationPipe } from '../../../../core/pipes/format-duration.pipe';
import { getSportLabel } from '../../../../core/i18n/sport-labels';

@Component({
  selector: 'app-route-segment-matches',
  imports: [TranslatePipe, RouterLink, FormatDistancePipe, FormatDurationPipe],
  templateUrl: './route-segment-matches.html',
  styleUrl: './route-segment-matches.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class RouteSegmentMatchesComponent {
  public readonly sportLabel = getSportLabel;
  private readonly coordinatorService = inject(WorkoutDetailCoordinatorService);
  public readonly matches = input.required<RouteSegmentMatch[]>();
  public readonly viewMode = input(false);

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
