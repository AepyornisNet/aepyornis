import {
  ChangeDetectionStrategy,
  Component,
  HostListener,
  inject,
  OnInit,
  signal,
} from '@angular/core';

import { TranslatePipe } from '@ngx-translate/core';
import { RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { Workout } from '../../../../core/types/workout';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { Api } from '../../../../core/services/api';

@Component({
  selector: 'app-recent-activity',
  imports: [RouterLink, AppIcon, TranslatePipe],
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
  public readonly likingWorkoutIDs = signal<Record<number, boolean>>({});

  public ngOnInit(): void {
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

  public isLiking(workoutID: number): boolean {
    return !!this.likingWorkoutIDs()[workoutID];
  }

  public async likeWorkout(workoutID: number): Promise<void> {
    const workout = this.displayedWorkouts().find((item) => item.id === workoutID);
    if (!workout || workout.liked_by_me || this.isLiking(workoutID)) {
      return;
    }

    this.likingWorkoutIDs.update((current) => ({ ...current, [workoutID]: true }));

    try {
      const response = await firstValueFrom(this.api.likeWorkout(workoutID));
      if (!response?.results) {
        return;
      }

      this.displayedWorkouts.update((current) =>
        current.map((item) => {
          if (item.id !== workoutID) {
            return item;
          }

          return {
            ...item,
            liked_by_me: response.results.liked,
            likes_count: response.results.likes_count,
          };
        }),
      );
    } catch (error) {
      console.error('Failed to like workout:', error);
    } finally {
      this.likingWorkoutIDs.update((current) => {
        const updated = { ...current };
        delete updated[workoutID];
        return updated;
      });
    }
  }

  @HostListener('window:scroll')
  public onWindowScroll(): void {
    const scrollPosition = window.pageYOffset + window.innerHeight;
    const pageHeight = document.documentElement.scrollHeight;
    const threshold = 300;

    if (pageHeight - scrollPosition < threshold && !this.loading() && this.hasMore()) {
      this.loadMore();
    }
  }
}
