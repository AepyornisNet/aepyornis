import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';

import { TranslatePipe } from '@ngx-translate/core';
import { RouterLink } from '@angular/router';
import { WorkoutRecord } from '../../../../core/types/workout';
import { FormatSpeedPipe } from '../../../../core/pipes/format-speed.pipe';
import { FormatDistancePipe } from '../../../../core/pipes/format-distance.pipe';
import { FormatElevationPipe } from '../../../../core/pipes/format-elevation.pipe';
import { FormatDurationPipe } from '../../../../core/pipes/format-duration.pipe';
import { FormatDatePipe } from '../../../../core/pipes/format-date.pipe';

@Component({
  selector: 'app-records',
  imports: [
    RouterLink,
    TranslatePipe,
    FormatSpeedPipe,
    FormatDistancePipe,
    FormatElevationPipe,
    FormatDurationPipe,
    FormatDatePipe,
  ],
  templateUrl: './records.html',
  styleUrl: './records.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Records {
  public readonly records = input<WorkoutRecord[]>([]);

  public readonly activeRecords = computed((): WorkoutRecord[] =>
    this.records().filter((r) => r.active && r.distance),
  );
}
