import { Component, signal, inject, OnInit, HostListener } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { Workout } from '../../../../core/types/workout';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { Api } from '../../../../core/services/api';

@Component({
  selector: 'app-recent-activity',
  imports: [CommonModule, RouterLink, AppIcon],
  templateUrl: './recent-activity.html',
  styleUrl: './recent-activity.scss'
})
export class RecentActivity implements OnInit {
  private api = inject(Api);

  displayedWorkouts = signal<Workout[]>([]);
  loading = signal(false);
  initialLoading = signal(true);
  hasMore = signal(true);
  readonly pageSize = 10;

  ngOnInit() {
    // Load initial workouts
    this.loadInitialWorkouts();
  }

  async loadInitialWorkouts() {
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

  formatDate(dateString: string): string {
    return new Date(dateString).toLocaleDateString();
  }

  formatDistance(distance: number): string {
    // Convert meters to kilometers
    return (distance / 1000).toFixed(2);
  }

  formatDuration(seconds: number): string {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    if (hours > 0) {
      return `${hours}h ${minutes}m`;
    }
    return `${minutes}m`;
  }

  formatWeight(weight: number): string {
    return weight.toFixed(1);
  }

  async loadMore() {
    if (this.loading() || !this.hasMore()) {
      return;
    }

    this.loading.set(true);
    try {
      const currentOffset = this.displayedWorkouts().length;
      const response = await firstValueFrom(this.api.getRecentWorkouts(this.pageSize, currentOffset));

      if (response?.results && response.results.length > 0) {
        this.displayedWorkouts.update(current => [...current, ...response.results]);
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
  onWindowScroll() {
    // Check if user has scrolled near the bottom of the page
    const scrollPosition = window.pageYOffset + window.innerHeight;
    const pageHeight = document.documentElement.scrollHeight;
    const threshold = 300; // pixels from bottom

    if (pageHeight - scrollPosition < threshold && !this.loading() && this.hasMore()) {
      this.loadMore();
    }
  }
}
