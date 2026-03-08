import { computed, inject, Injectable, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';
import { Api } from '../../../core/services/api';
import { WorkoutDetail, WorkoutLike } from '../../../core/types/workout';

/**
 * Service responsible for managing workout data and providing common formatting utilities.
 */
@Injectable({
  providedIn: 'root',
})
export class WorkoutDetailDataService {
  private api = inject(Api);

  public readonly workout = signal<WorkoutDetail | null>(null);
  public readonly likes = signal<WorkoutLike[]>([]);
  public readonly likesLoading = signal(false);
  public readonly loading = signal(false);
  public readonly error = signal<string | null>(null);

  // Computed values
  public readonly hasTrack = computed(() => {
    const w = this.workout();
    return !!(w?.has_location_data ?? w?.has_tracks);
  });

  public readonly hasChartData = computed(() => {
    return this.hasTimeChartData() || this.hasDistanceChartData();
  });

  public readonly hasTimeChartData = computed(() => {
    const d = this.workout()?.map_data?.details;
    if (!d || d.time.length === 0) {
      return false;
    }

    return (
      this.hasNonZeroSeries(d.speed) ||
      this.hasNonZeroSeries(d.elevation) ||
      this.hasAnyNumericExtraMetric(d.extra_metrics)
    );
  });

  public readonly hasDistanceChartData = computed(() => {
    const d = this.workout()?.map_data?.details;
    if (!d || d.time.length === 0 || d.distance.length === 0) {
      return false;
    }

    const hasDistanceValues = d.distance.some((value) => Number.isFinite(value) && value > 0);
    if (!hasDistanceValues) {
      return false;
    }

    return (
      this.hasNonZeroSeries(d.speed) ||
      this.hasNonZeroSeries(d.elevation) ||
      this.hasAnyNumericExtraMetric(d.extra_metrics)
    );
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

  public readonly hasPowerData = computed(() => {
    const metrics = this.workout()?.map_data?.details?.extra_metrics?.['power'];
    return Array.isArray(metrics) && metrics.some((value) => typeof value === 'number');
  });

  public readonly hasZoneCharts = computed(
    () => this.hasHeartRateDistribution() || this.hasPowerDistribution(),
  );

  public async loadWorkout(id: number): Promise<void> {
    this.loading.set(true);
    this.error.set(null);

    try {
      const response = await firstValueFrom(this.api.getWorkout(id));

      if (response) {
        this.workout.set(response.results);
        void this.loadWorkoutLikes(response.results.id);
      }
    } catch (err) {
      console.error('Failed to load workout:', err);
      this.error.set('Failed to load workout. Please try again.');
    } finally {
      this.loading.set(false);
    }
  }

  public async loadWorkoutLikes(workoutId: number): Promise<void> {
    if (!workoutId) {
      this.likes.set([]);
      return;
    }

    this.likesLoading.set(true);
    try {
      const response = await firstValueFrom(this.api.getWorkoutLikes(workoutId));
      this.likes.set(response?.results || []);
    } catch (err) {
      console.error('Failed to load workout likes:', err);
      this.likes.set([]);
    } finally {
      this.likesLoading.set(false);
    }
  }

  public clearWorkout(): void {
    this.workout.set(null);
    this.likes.set([]);
    this.likesLoading.set(false);
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

  private hasNonZeroSeries(values: (number | null | undefined)[] | undefined): boolean {
    if (!Array.isArray(values)) {
      return false;
    }

    return values.some(
      (value) => typeof value === 'number' && Number.isFinite(value) && Math.abs(value) > 0,
    );
  }

  private hasAnyNumericExtraMetric(
    extraMetrics: Record<string, (number | null)[]> | undefined,
  ): boolean {
    if (!extraMetrics) {
      return false;
    }

    return Object.values(extraMetrics).some(
      (arr) =>
        Array.isArray(arr) &&
        arr.some((value) => typeof value === 'number' && Number.isFinite(value)),
    );
  }
}
