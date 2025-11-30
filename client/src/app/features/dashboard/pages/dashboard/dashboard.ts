import { ChangeDetectionStrategy, Component, inject, OnInit, signal } from '@angular/core';

import { firstValueFrom } from 'rxjs';
import { Api } from '../../../../core/services/api';
import { Totals, WorkoutRecord } from '../../../../core/types/workout';
import { WorkoutCalendar } from '../../../workouts/components/workout-calendar/workout-calendar';
import { KeyMetrics } from '../../components/key-metrics/key-metrics';
import { Records } from '../../components/records/records';
import { RecentActivity } from '../../components/recent-activity/recent-activity';
import { TranslatePipe } from '@ngx-translate/core';

@Component({
  selector: 'app-dashboard',
  imports: [WorkoutCalendar, KeyMetrics, Records, RecentActivity, TranslatePipe],
  templateUrl: './dashboard.html',
  styleUrl: './dashboard.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Dashboard implements OnInit {
  private api = inject(Api);

  public readonly totals = signal<Totals | null>(null);
  public readonly records = signal<WorkoutRecord[]>([]);
  public readonly loading = signal(true);
  public readonly error = signal<string | null>(null);

  public ngOnInit(): void {
    this.loadDashboardData();
  }

  public async loadDashboardData(): Promise<void> {
    this.loading.set(true);
    this.error.set(null);

    try {
      // Load totals and records in parallel
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
      console.error('Failed to load dashboard data:', err);
      this.error.set('Failed to load dashboard data. Please try again.');
    } finally {
      this.loading.set(false);
    }
  }
}
