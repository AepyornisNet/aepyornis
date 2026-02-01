import { ChangeDetectionStrategy, Component, inject, OnInit, signal } from '@angular/core';

import { FormBuilder, FormGroup, ReactiveFormsModule } from '@angular/forms';
import { firstValueFrom, Observable } from 'rxjs';
import { Api } from '../../../../core/services/api';
import { Statistics as StatisticsData } from '../../../../core/types/statistics';
import { UserPreferredUnits } from '../../../../core/types/user';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { StatisticChartComponent } from '../../components/statistic-chart/statistic-chart';
import { StatisticsNav } from '../../components/statistics-nav/statistics-nav';
import { TranslateService, Translation } from '@ngx-translate/core';
import { AsyncPipe } from '@angular/common';
import { TranslatePipe } from '@ngx-translate/core';

type StatisticOption = {
  key: string;
  label: Observable<Translation>;
};

@Component({
  selector: 'app-statistics',
  imports: [ReactiveFormsModule, AppIcon, StatisticChartComponent, StatisticsNav, AsyncPipe, TranslatePipe],
  templateUrl: './statistics.html',
  styleUrl: './statistics.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Statistics implements OnInit {
  private api = inject(Api);
  private fb = inject(FormBuilder);
  private translate = inject(TranslateService);

  public readonly statistics = signal<StatisticsData | null>(null);
  public readonly preferredUnits = signal<UserPreferredUnits | null>(null);
  public readonly loading = signal(true);
  public readonly error = signal<string | null>(null);

  // Reactive form for filters
  public filterForm!: FormGroup;

  public sinceOptions: StatisticOption[] = [
    { key: '7 days', label: this.translate.stream('{{num}} days', { num : 7 }) },
    { key: '1 month', label: this.translate.stream('1 month') },
    { key: '3 months', label: this.translate.stream('{{num}} months', { num: 3 }) },
    { key: '6 months', label: this.translate.stream('{{num}} months', { num: 6 }) },
    { key: '1 year', label: this.translate.stream('1 year') },
    { key: '2 years', label: this.translate.stream('{{num}} years', { num: 2 }) },
    { key: '5 years', label: this.translate.stream('{{num}} years', { num: 5 }) },
    { key: '10 years', label: this.translate.stream('{{num}} years', { num: 10 }) },
    { key: 'forever', label: this.translate.stream('forever') },
  ];

  public perOptions: StatisticOption[] = [
    { key: 'day', label: this.translate.stream('Day') },
    { key: 'week', label: this.translate.stream('Week') },
    { key: 'month', label: this.translate.stream('Month') },
    { key: 'year', label: this.translate.stream('Year') },
  ];

  public ngOnInit(): void {
    // Initialize filter form
    this.filterForm = this.fb.group({
      since: ['1 year'],
      per: ['month'],
    });

    this.loadPreferredUnits();
    this.loadStatistics();
  }

  public async loadPreferredUnits(): Promise<void> {
    try {
      const profile = await firstValueFrom(this.api.getProfile());
      if (profile?.results?.profile?.preferred_units) {
        this.preferredUnits.set(profile.results.profile.preferred_units);
      }
    } catch (err) {
      console.error('Failed to load preferred units:', err);
    }
  }

  public async loadStatistics(): Promise<void> {
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

  public onFilterChange(): void {
    this.loadStatistics();
  }
}
