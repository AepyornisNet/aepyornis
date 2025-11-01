import { computed, inject, Injectable, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';
import { Api } from '../../../core/services/api';
import { WorkoutDetail } from '../../../core/types/workout';

/**
 * Service responsible for managing workout data and providing common formatting utilities.
 */
@Injectable({
  providedIn: 'root',
})
export class WorkoutDetailDataService {
  private api = inject(Api);

  readonly workout = signal<WorkoutDetail | null>(null);
  readonly loading = signal(false);
  readonly error = signal<string | null>(null);

  // Computed values
  readonly hasMapData = computed(() => {
    const w = this.workout();
    return w?.map_data?.details?.position && w.map_data.details.position.length > 0;
  });

  readonly hasClimbs = computed(() => {
    const w = this.workout();
    return w?.climbs && w.climbs.length > 0;
  });

  readonly hasRouteSegmentMatches = computed(() => {
    const w = this.workout();
    return w?.route_segment_matches && w.route_segment_matches.length > 0;
  });

  readonly extraMetrics = computed(() => {
    const w = this.workout();
    return w?.map_data?.extra_metrics || [];
  });

  async loadWorkout(id: number): Promise<void> {
    this.loading.set(true);
    this.error.set(null);

    try {
      const response = await firstValueFrom(this.api.getWorkout(id));

      if (response) {
        this.workout.set(response.results);
      }
    } catch (err) {
      console.error('Failed to load workout:', err);
      this.error.set('Failed to load workout. Please try again.');
    } finally {
      this.loading.set(false);
    }
  }

  clearWorkout(): void {
    this.workout.set(null);
    this.loading.set(false);
    this.error.set(null);
  }

  // Formatting utilities
  formatDate(dateString: string): string {
    return new Date(dateString).toLocaleString();
  }

  formatDistance(distance: number): string {
    return (distance / 1000).toFixed(2);
  }

  formatDuration(seconds: number): string {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const secs = Math.floor(seconds % 60);

    if (hours > 0) {
      return `${hours}h ${minutes}m ${secs}s`;
    }
    if (minutes > 0) {
      return `${minutes}m ${secs}s`;
    }
    return `${secs}s`;
  }

  formatElevation(elevation: number): string {
    return elevation.toFixed(1);
  }

  formatSpeed(speed: number): string {
    return (speed * 3.6).toFixed(2); // Convert m/s to km/h
  }
}
