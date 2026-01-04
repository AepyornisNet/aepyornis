import { ChangeDetectionStrategy, Component, computed, inject, OnInit, signal } from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { TranslatePipe } from '@ngx-translate/core';
import { firstValueFrom } from 'rxjs';

import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { Pagination } from '../../../../core/components/pagination/pagination';
import { Api } from '../../../../core/services/api';
import { UserPreferredUnits } from '../../../../core/types/user';
import { ClimbRecordEntry } from '../../../../core/types/workout';

const METERS_TO_MILES = 0.000621371;
const METERS_TO_FEET = 3.28084;

@Component({
  selector: 'app-climb-ranking',
  standalone: true,
  imports: [RouterLink, AppIcon, Pagination, TranslatePipe],
  templateUrl: './climb-ranking.html',
  styleUrl: './climb-ranking.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ClimbRankingPage implements OnInit {
  private api = inject(Api);
  private route = inject(ActivatedRoute);

  public readonly loading = signal(true);
  public readonly error = signal<string | null>(null);
  public readonly records = signal<ClimbRecordEntry[]>([]);
  public readonly preferredUnits = signal<UserPreferredUnits | null>(null);
  public readonly page = signal(1);
  public readonly totalPages = signal(1);
  public readonly totalCount = signal(0);
  public readonly perPage = 20;

  public workoutType = '';

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

  public async ngOnInit(): Promise<void> {
    this.workoutType = this.route.snapshot.paramMap.get('workoutType') ?? '';
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
          this.api.getClimbRanking({
            workout_type: this.workoutType,
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
      console.error('Failed to load climb ranking', err);
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

  public formatElevation(meters?: number | null): string {
    if (meters === undefined || meters === null) {
      return '—';
    }

    const units = this.preferredUnits();
    if (!units || units.elevation === 'm') {
      return `${meters.toFixed(0)} m`;
    }

    return `${(meters * METERS_TO_FEET).toFixed(0)} ft`;
  }

  public formatDistance(meters?: number | null): string {
    if (meters === undefined || meters === null) {
      return '—';
    }

    const units = this.preferredUnits();
    if (!units || units.distance === 'km') {
      return `${(meters / 1000).toFixed(2)} km`;
    }

    return `${(meters * METERS_TO_MILES).toFixed(2)} mi`;
  }

  public formatSlope(slope?: number | null): string {
    if (slope === undefined || slope === null) {
      return '—';
    }

    return `${(slope * 100).toFixed(1)}%`;
  }

  public formatDate(date: string | undefined): string {
    if (!date) {
      return '';
    }
    return new Date(date).toLocaleDateString();
  }
}
