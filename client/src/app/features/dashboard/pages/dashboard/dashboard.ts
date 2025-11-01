import { Component, inject, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
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
  imports: [CommonModule, WorkoutCalendar, KeyMetrics, Records, RecentActivity, TranslatePipe],
  templateUrl: './dashboard.html',
  styleUrl: './dashboard.scss',
})
export class Dashboard implements OnInit {
  private api = inject(Api);

  readonly totals = signal<Totals | null>(null);
  readonly records = signal<WorkoutRecord[]>([]);
  readonly loading = signal(true);
  readonly error = signal<string | null>(null);

  ngOnInit() {
    this.loadDashboardData();
  }

  async loadDashboardData() {
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
