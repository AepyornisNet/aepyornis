import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';

import { TranslatePipe } from '@ngx-translate/core';
import { RouterLink } from '@angular/router';
import { WorkoutRecord } from '../../../../core/types/workout';

@Component({
  selector: 'app-records',
  imports: [RouterLink, TranslatePipe],
  templateUrl: './records.html',
  styleUrl: './records.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Records {
  public readonly records = input<WorkoutRecord[]>([]);

  public readonly activeRecords = computed((): WorkoutRecord[] =>
    this.records().filter((r) => r.active && r.distance),
  );

  public formatDate(dateString: string): string {
    return new Date(dateString).toLocaleDateString();
  }

  public formatSpeed(speed: number): string {
    return (speed * 3.6).toFixed(2);
  }

  public formatDistance(distance: number): string {
    return (distance / 1000).toFixed(2);
  }

  public formatElevation(elevation: number): string {
    return elevation.toFixed(0);
  }

  public formatDuration(seconds: number): string {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    if (hours > 0) {
      return `${hours}h ${minutes}m`;
    }
    return `${minutes}m`;
  }
}
