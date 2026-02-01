import { ChangeDetectionStrategy, Component, computed, inject, OnInit, signal } from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { Api } from '../../../../core/services/api';
import { DistanceRecordEntry } from '../../../../core/types/workout';
import { UserPreferredUnits } from '../../../../core/types/user';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { Pagination } from '../../../../core/components/pagination/pagination';
import { TranslatePipe } from '@ngx-translate/core';
import { getSportLabel } from '../../../../core/i18n/sport-labels';

@Component({
  selector: 'app-records-ranking',
  standalone: true,
  imports: [RouterLink, AppIcon, Pagination, TranslatePipe],
  templateUrl: './records-ranking.html',
  styleUrl: './records-ranking.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class RecordsRankingPage implements OnInit {
  private api = inject(Api);
  private route = inject(ActivatedRoute);

  private readonly metersToMiles = 0.000621371;
  private readonly metersToFeet = 3.28084;

  public readonly loading = signal(true);
  public readonly error = signal<string | null>(null);
  public readonly records = signal<DistanceRecordEntry[]>([]);
  public readonly preferredUnits = signal<UserPreferredUnits | null>(null);
  public readonly page = signal(1);
  public readonly totalPages = signal(1);
  public readonly totalCount = signal(0);
  public readonly perPage = 20;
  public readonly sportLabel = getSportLabel;

  private readonly visiblePages = computed(() => {
    const current = this.page();
    const total = this.totalPages();
    const maxVisible = 7;

    if (total <= maxVisible) {
      return Array.from({ length: total }, (_, i) => i + 1);
    }

    const pages: number[] = [1];
    let start = Math.max(2, current - 2);
    let end = Math.min(total - 1, current + 2);

    if (current <= 3) {
      end = Math.min(total - 1, 5);
    }

    if (current >= total - 2) {
      start = Math.max(2, total - 4);
    }

    if (start > 2) {
      pages.push(-1);
    }

    for (let i = start; i <= end; i++) {
      pages.push(i);
    }

    if (end < total - 1) {
      pages.push(-1);
    }

    if (total > 1) {
      pages.push(total);
    }

    return pages;
  });

  public workoutType = '';
  public label = '';

  public async ngOnInit(): Promise<void> {
    const workoutType = this.route.snapshot.paramMap.get('workoutType') || '';
    const label = this.route.snapshot.paramMap.get('label') || '';

    this.workoutType = workoutType;
    this.label = decodeURIComponent(label);

    await this.loadData();
  }

  private async loadData(page?: number): Promise<void> {
    this.loading.set(true);
    this.error.set(null);
    const targetPage = page ?? this.page();
    this.page.set(targetPage);

    try {
      const [profile, ranking] = await Promise.all([
        firstValueFrom(this.api.getProfile()),
        firstValueFrom(
          this.api.getDistanceRecordRanking({
            workout_type: this.workoutType,
            label: this.label,
            page: targetPage,
            per_page: this.perPage,
          }),
        ),
      ]);

      if (profile?.results?.profile?.preferred_units) {
        this.preferredUnits.set(profile.results.profile.preferred_units);
      }

      this.records.set(ranking.results ?? []);
      this.totalPages.set(ranking.total_pages ?? 1);
      this.totalCount.set(ranking.total_count ?? 0);
    } catch (err) {
      console.error('Failed to load distance ranking', err);
      this.error.set('Failed to load ranking. Please try again.');
    } finally {
      this.loading.set(false);
    }
  }

  public rankNumber(index: number): number {
    return (this.page() - 1) * this.perPage + index + 1;
  }

  public paginationSource(): {
    current: () => number;
    total: () => number;
    pages: () => number[];
    hasPrevious: () => boolean;
    hasNext: () => boolean;
    totalCount: () => number;
    previous: () => void;
    goTo: (page: number) => void;
    next: () => void;
  } {
    return {
      current: () => this.page(),
      total: () => this.totalPages(),
      pages: () => this.visiblePages(),
      hasPrevious: () => this.page() > 1,
      hasNext: () => this.page() < this.totalPages(),
      totalCount: () => this.totalCount(),
      previous: (): void => {
        void this.goToPage(this.page() - 1);
      },
      goTo: (p: number): void => {
        void this.goToPage(p);
      },
      next: (): void => {
        void this.goToPage(this.page() + 1);
      },
    };
  }

  private async goToPage(page: number): Promise<void> {
    if (page < 1 || page > this.totalPages()) {
      return;
    }
    await this.loadData(page);
  }

  public formatDistance(meters: number | undefined): string {
    if (meters === undefined || meters === null) {
      return '—';
    }

    const units = this.preferredUnits();
    if (!units || units.distance === 'km') {
      return `${(meters / 1000).toFixed(2)} km`;
    }

    return `${(meters * this.metersToMiles).toFixed(2)} mi`;
  }

  public formatElevation(meters: number | undefined): string {
    if (meters === undefined || meters === null) {
      return '—';
    }

    const units = this.preferredUnits();
    if (!units || units.elevation === 'm') {
      return `${meters.toFixed(0)} m`;
    }

    return `${(meters * this.metersToFeet).toFixed(0)} ft`;
  }

  public formatSpeed(metersPerSecond: number | undefined): string {
    if (!metersPerSecond && metersPerSecond !== 0) {
      return '—';
    }

    const units = this.preferredUnits();
    if (!units || units.speed === 'km/h') {
      return `${(metersPerSecond * 3.6).toFixed(2)} km/h`;
    }

    return `${(metersPerSecond * 2.23694).toFixed(2)} mph`;
  }

  public formatDuration(seconds: number | undefined): string {
    if (seconds === undefined || seconds === null) {
      return '—';
    }

    const rounded = Math.round(seconds);
    const hours = Math.floor(rounded / 3600);
    const minutes = Math.floor((rounded % 3600) / 60);
    const secs = rounded % 60;

    const parts: string[] = [];
    if (hours > 0) {
      parts.push(`${hours}h`);
    }
    if (minutes > 0) {
      parts.push(`${minutes}m`);
    }
    if (hours === 0 && minutes === 0) {
      parts.push(`${secs}s`);
    }

    return parts.join(' ');
  }

  public formatDate(date: string | undefined): string {
    if (!date) {
      return '';
    }
    return new Date(date).toLocaleDateString();
  }
}
