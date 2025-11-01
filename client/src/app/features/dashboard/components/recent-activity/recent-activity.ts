import {
  ChangeDetectionStrategy,
  Component,
  HostListener,
  inject,
  OnInit,
  signal,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { TranslatePipe } from '@ngx-translate/core';
import { RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { Workout } from '../../../../core/types/workout';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { Api } from '../../../../core/services/api';

@Component({
  selector: 'app-recent-activity',
  imports: [CommonModule, RouterLink, AppIcon, TranslatePipe],
  templateUrl: './recent-activity.html',
  styleUrl: './recent-activity.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class RecentActivity implements OnInit {
  private api = inject(Api);

  public readonly displayedWorkouts = signal<Workout[]>([]);
  public readonly loading = signal(false);
  public readonly initialLoading = signal(true);
  public readonly hasMore = signal(true);
  public readonly pageSize = 10;

  public ngOnInit(): void {
    // Load initial workouts
    this.loadInitialWorkouts();
  }

  public async loadInitialWorkouts(): Promise<void> {
    this.initialLoading.set(true);
    try {
      const response = await firstValueFrom(this.api.getRecentWorkouts(this.pageSize, 0));
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

  public formatDate(dateString: string): string {
    return new Date(dateString).toLocaleDateString();
  }

  public formatDistance(distance: number): string {
    // Convert meters to kilometers
    return (distance / 1000).toFixed(2);
  }

  public formatDuration(seconds: number): string {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    if (hours > 0) {
      return `${hours}h ${minutes}m`;
    }
    return `${minutes}m`;
  }

  public formatWeight(weight: number): string {
    return weight.toFixed(1);
  }

  public async loadMore(): Promise<void> {
    if (this.loading() || !this.hasMore()) {
      return;
    }

    this.loading.set(true);
    try {
      const currentOffset = this.displayedWorkouts().length;
      const response = await firstValueFrom(
        this.api.getRecentWorkouts(this.pageSize, currentOffset),
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

  @HostListener('window:scroll')
  public onWindowScroll(): void {
    // Check if user has scrolled near the bottom of the page
    const scrollPosition = window.pageYOffset + window.innerHeight;
    const pageHeight = document.documentElement.scrollHeight;
    const threshold = 300; // pixels from bottom

    if (pageHeight - scrollPosition < threshold && !this.loading() && this.hasMore()) {
      this.loadMore();
    }
  }
}
