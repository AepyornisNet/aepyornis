import { ChangeDetectionStrategy, Component, inject, OnInit, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { Api } from '../../../../core/services/api';
import { WorkoutRecord } from '../../../../core/types/workout';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { StatisticsNav } from '../../components/statistics-nav/statistics-nav';
import { TranslatePipe } from '@ngx-translate/core';
import { getSportLabel } from '../../../../core/i18n/sport-labels';
import { FormatDistancePipe } from '../../../../core/pipes/format-distance.pipe';
import { FormatDurationPipe } from '../../../../core/pipes/format-duration.pipe';
import { FormatElevationPipe } from '../../../../core/pipes/format-elevation.pipe';
import { FormatSpeedPipe } from '../../../../core/pipes/format-speed.pipe';
import { FormatDatePipe } from '../../../../core/pipes/format-date.pipe';

@Component({
  selector: 'app-statistics-records',
  standalone: true,
  imports: [
    RouterLink,
    AppIcon,
    StatisticsNav,
    TranslatePipe,
    FormatDatePipe,
    FormatDistancePipe,
    FormatDurationPipe,
    FormatSpeedPipe,
    FormatElevationPipe,
  ],
  templateUrl: './records.html',
  styleUrl: './records.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class StatisticsRecords implements OnInit {
  private api = inject(Api);

  public readonly records = signal<WorkoutRecord[]>([]);
  public readonly loading = signal(true);
  public readonly error = signal<string | null>(null);
  public readonly sportLabel = getSportLabel;

  public async ngOnInit(): Promise<void> {
    await this.loadData();
  }

  private async loadData(): Promise<void> {
    this.loading.set(true);
    this.error.set(null);

    try {
      const records = await firstValueFrom(this.api.getRecords());

      if (records?.results) {
        this.records.set(records.results);
      }
    } catch (err) {
      console.error('Failed to load statistics records:', err);
      this.error.set('Failed to load records. Please try again.');
    } finally {
      this.loading.set(false);
    }
  }

  public hasDistanceRecords(record: WorkoutRecord): boolean {
    return Boolean(record.distance_records && record.distance_records.length > 0);
  }
}
