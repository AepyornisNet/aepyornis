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

  public readonly workout = signal<WorkoutDetail | null>(null);
  public readonly loading = signal(false);
  public readonly error = signal<string | null>(null);

  // Computed values
  public readonly hasMapData = computed(() => {
    const w = this.workout();
    return w?.map_data?.details?.position && w.map_data.details.position.length > 0;
  });

  public readonly hasClimbs = computed(() => {
    const w = this.workout();
    return w?.climbs && w.climbs.length > 0;
  });

  public readonly hasRouteSegmentMatches = computed(() => {
    const w = this.workout();
    return w?.route_segment_matches && w.route_segment_matches.length > 0;
  });

  public readonly hasRecords = computed(() => {
    const w = this.workout();
    return w?.records && w.records.length > 0;
  });

  public readonly extraMetrics = computed(() => {
    const w = this.workout();
    return w?.map_data?.extra_metrics || [];
  });

  public readonly hasHeartRateDistribution = computed(() => {
    const metrics = this.workout()?.map_data?.details?.extra_metrics?.['hr-zone'];
    return Array.isArray(metrics) && metrics.some((value) => typeof value === 'number');
  });

  public readonly hasPowerDistribution = computed(() => {
    const metrics = this.workout()?.map_data?.details?.extra_metrics?.['zone'];
    return Array.isArray(metrics) && metrics.some((value) => typeof value === 'number');
  });

  public readonly hasZoneCharts = computed(() =>
    this.hasHeartRateDistribution() || this.hasPowerDistribution(),
  );

  public async loadWorkout(id: number): Promise<void> {
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

  public clearWorkout(): void {
    this.workout.set(null);
    this.loading.set(false);
    this.error.set(null);
  }

  // Formatting utilities
  public formatDate(dateString: string): string {
    return new Date(dateString).toLocaleString();
  }

  public formatDistance(distance: number): string {
    return (distance / 1000).toFixed(2);
  }

  public formatDuration(seconds: number): string {
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

  public formatElevation(elevation: number): string {
    return elevation.toFixed(1);
  }

  public formatSpeed(speed: number): string {
    return (speed * 3.6).toFixed(2); // Convert m/s to km/h
  }
}
