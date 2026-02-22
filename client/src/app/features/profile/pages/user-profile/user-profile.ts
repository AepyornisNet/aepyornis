import { ChangeDetectionStrategy, Component, inject, OnInit, signal } from '@angular/core';

import { firstValueFrom } from 'rxjs';
import { Api } from '../../../../core/services/api';
import { Totals, WorkoutRecord } from '../../../../core/types/workout';
import { WorkoutCalendar } from '../../components/workout-calendar/workout-calendar';
import { KeyMetrics } from '../../components/key-metrics/key-metrics';
import { Records } from '../../components/records/records';
import { TranslatePipe, TranslateService } from '@ngx-translate/core';

@Component({
  selector: 'app-user-profile',
  imports: [WorkoutCalendar, KeyMetrics, Records, TranslatePipe],
  templateUrl: './user-profile.html',
  styleUrl: './user-profile.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class UserProfile implements OnInit {
  private api = inject(Api);
  private translate = inject(TranslateService);

  public readonly totals = signal<Totals | null>(null);
  public readonly records = signal<WorkoutRecord[]>([]);
  public readonly loading = signal(true);
  public readonly error = signal<string | null>(null);

  public ngOnInit(): void {
    this.loadProfileData();
  }

  public async loadProfileData(): Promise<void> {
    this.loading.set(true);
    this.error.set(null);

    try {
      const [totalsResponse, recordsResponse] = await Promise.all([
        firstValueFrom(this.api.getTotals()),
        firstValueFrom(this.api.getRecords()),
      ]);

      if (totalsResponse) {
        this.totals.set(totalsResponse.results);
      }

      if (recordsResponse) {
        this.records.set(recordsResponse.results);
      }
    } catch (err) {
      console.error('Failed to load profile data:', err);
      this.error.set(
        this.translate.instant('Failed to load {{page}} data. Please try again.', {
          page: this.translate.instant('profile'),
        }),
      );
    } finally {
      this.loading.set(false);
    }
  }
}
