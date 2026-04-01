import { ChangeDetectionStrategy, Component, effect, inject, input, signal } from '@angular/core';

import { TranslatePipe } from '@ngx-translate/core';
import { firstValueFrom } from 'rxjs';
import { Workout } from '../../../../core/types/workout';
import { Api } from '../../../../core/services/api';
import { FeedPost } from '../feed-post/feed-post';

@Component({
  selector: 'app-recent-activity',
  imports: [FeedPost, TranslatePipe],
  templateUrl: './recent-activity.html',
  styleUrl: './recent-activity.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
  host: {
    '(window:scroll)': 'onWindowScroll()',
  },
})
export class RecentActivity {
  private api = inject(Api);
  public readonly feedScope = input<'following' | 'global'>('following');

  public readonly displayedWorkouts = signal<Workout[]>([]);
  public readonly loading = signal(false);
  public readonly initialLoading = signal(true);
  public readonly hasMore = signal(true);
  public readonly pageSize = 10;

  public constructor() {
    effect(() => {
      this.feedScope();
      this.displayedWorkouts.set([]);
      this.hasMore.set(true);
      this.loading.set(false);
      void this.loadInitialWorkouts();
    });
  }

  public async loadInitialWorkouts(): Promise<void> {
    this.initialLoading.set(true);
    try {
      const response = await firstValueFrom(
        this.api.getRecentWorkouts(this.pageSize, 0, this.feedScope()),
      );
      if (response?.results) {
        this.displayedWorkouts.set(response.results);
        this.hasMore.set(response.results.length === this.pageSize);
      }
    } catch (error) {
      console.error('Failed to load initial workouts:', error);
    } finally {
      this.initialLoading.set(false);
    }
  }

  public async loadMore(): Promise<void> {
    if (this.loading() || !this.hasMore()) {
      return;
    }

    this.loading.set(true);
    try {
      const currentOffset = this.displayedWorkouts().length;
      const response = await firstValueFrom(
        this.api.getRecentWorkouts(this.pageSize, currentOffset, this.feedScope()),
      );

      if (response?.results && response.results.length > 0) {
        this.displayedWorkouts.update((current) => [...current, ...response.results]);
        this.hasMore.set(response.results.length === this.pageSize);
      } else {
        this.hasMore.set(false);
      }
    } catch (error) {
      console.error('Failed to load more workouts:', error);
      this.hasMore.set(false);
    } finally {
      this.loading.set(false);
    }
  }

  public onWindowScroll(): void {
    const scrollPosition = window.pageYOffset + window.innerHeight;
    const pageHeight = document.documentElement.scrollHeight;
    const threshold = 300;

    if (pageHeight - scrollPosition < threshold && !this.loading() && this.hasMore()) {
      void this.loadMore();
    }
  }
}
