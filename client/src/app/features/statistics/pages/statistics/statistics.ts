import { ChangeDetectionStrategy, Component, inject, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, FormGroup, ReactiveFormsModule } from '@angular/forms';
import { firstValueFrom } from 'rxjs';
import { Api } from '../../../../core/services/api';
import { Statistics as StatisticsData } from '../../../../core/types/statistics';
import { UserPreferredUnits } from '../../../../core/types/user';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { StatisticChartComponent } from '../../components/statistic-chart/statistic-chart';

interface StatisticOption {
  key: string;
  label: string;
}

@Component({
  selector: 'app-statistics',
  imports: [CommonModule, ReactiveFormsModule, AppIcon, StatisticChartComponent],
  templateUrl: './statistics.html',
  styleUrl: './statistics.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Statistics implements OnInit {
  private api = inject(Api);
  private fb = inject(FormBuilder);

  readonly statistics = signal<StatisticsData | null>(null);
  readonly preferredUnits = signal<UserPreferredUnits | null>(null);
  readonly loading = signal(true);
  readonly error = signal<string | null>(null);

  // Reactive form for filters
  filterForm!: FormGroup;

  sinceOptions: StatisticOption[] = [
    { key: '7 day', label: '7 days' },
    { key: '1 month', label: '1 month' },
    { key: '3 months', label: '3 months' },
    { key: '6 months', label: '6 months' },
    { key: '1 year', label: '1 year' },
    { key: '2 years', label: '2 years' },
    { key: '5 years', label: '5 years' },
    { key: '10 years', label: '10 years' },
    { key: 'forever', label: 'Forever' },
  ];

  perOptions: StatisticOption[] = [
    { key: 'day', label: 'Day' },
    { key: 'week', label: 'Week' },
    { key: 'month', label: 'Month' },
  ];

  ngOnInit() {
    // Initialize filter form
    this.filterForm = this.fb.group({
      since: ['1 year'],
      per: ['month'],
    });

    this.loadPreferredUnits();
    this.loadStatistics();
  }

  async loadPreferredUnits() {
    try {
      const profile = await firstValueFrom(this.api.getProfile());
      if (profile?.results?.profile?.preferred_units) {
        this.preferredUnits.set(profile.results.profile.preferred_units);
      }
    } catch (err) {
      console.error('Failed to load preferred units:', err);
    }
  }

  async loadStatistics() {
    this.loading.set(true);
    this.error.set(null);

    try {
      const formValue = this.filterForm.value;
      const response = await firstValueFrom(
        this.api.getStatistics({
          since: formValue.since,
          per: formValue.per,
        }),
      );

      if (response?.results) {
        this.statistics.set(response.results);
      }
    } catch (err) {
      console.error('Failed to load statistics:', err);
      this.error.set('Failed to load statistics. Please try again.');
    } finally {
      this.loading.set(false);
    }
  }

  onFilterChange() {
    this.loadStatistics();
  }
}
