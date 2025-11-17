import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { TranslatePipe } from '@ngx-translate/core';
import { ClimbSegment } from '../../../../core/types/workout';

@Component({
  selector: 'app-workout-climbs',
  imports: [CommonModule, TranslatePipe],
  templateUrl: './workout-climbs.html',
  styleUrl: './workout-climbs.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class WorkoutClimbsComponent {
  public readonly climbs = input.required<ClimbSegment[]>();

  public formatDistance(meters: number): string {
    return (meters / 1000).toFixed(2);
  }

  public formatElevation(meters: number): string {
    return meters.toFixed(0);
  }
}
