import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  OnInit,
  signal,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { firstValueFrom } from 'rxjs';
import { Api } from '../../../../core/services/api';
import { WorkoutDetail } from '../../../../core/types/workout';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { TranslatePipe } from '@ngx-translate/core';

@Component({
  selector: 'app-create-route-segment',
  imports: [CommonModule, FormsModule, AppIcon, TranslatePipe],
  templateUrl: './create-route-segment.html',
  styleUrl: './create-route-segment.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class CreateRouteSegmentPage implements OnInit {
  private api = inject(Api);
  private route = inject(ActivatedRoute);
  private router = inject(Router);

  public readonly workout = signal<WorkoutDetail | null>(null);
  public readonly loading = signal(true);
  public readonly error = signal<string | null>(null);
  public readonly creating = signal(false);

  // Form fields
  public readonly name = signal('');
  public readonly start = signal(1);
  public readonly end = signal(1);

  // Computed values
  public readonly totalPoints = computed(() => {
    const w = this.workout();
    return w?.map_data?.details?.position?.length || 0;
  });

  public readonly selectedDistance = computed(() => {
    const w = this.workout();
    const startIdx = this.start() - 1;
    const endIdx = this.end() - 1;

    if (!w?.map_data?.details?.distance || startIdx < 0 || endIdx < 0) {
      return 0;
    }

    const distances = w.map_data.details.distance;
    if (endIdx >= distances.length || startIdx >= distances.length) {
      return 0;
    }

    return Math.abs(distances[endIdx] - distances[startIdx]);
  });

  public ngOnInit(): void {
    this.route.params.subscribe((params) => {
      const id = parseInt(params['id']);
      if (id) {
        this.loadWorkout(id);
      }
    });
  }

  public async loadWorkout(id: number): Promise<void> {
    this.loading.set(true);
    this.error.set(null);

    try {
      const response = await firstValueFrom(this.api.getWorkout(id));

      if (response) {
        const workout = response.results;
        this.workout.set(workout);
        this.name.set(workout.name);

        // Set end to the last point
        const points = workout.map_data?.details?.position?.length || 1;
        this.end.set(points);
      }
    } catch (err) {
      console.error('Failed to load workout:', err);
      this.error.set('Failed to load workout. Please try again.');
    } finally {
      this.loading.set(false);
    }
  }

  public updateStart(value: number): void {
    this.start.set(value);

    // Ensure start is not greater than end
    if (value > this.end()) {
      this.end.set(value);
    }
  }

  public updateEnd(value: number): void {
    this.end.set(value);

    // Ensure end is not less than start
    if (value < this.start()) {
      this.start.set(value);
    }
  }

  public async createRouteSegment(): Promise<void> {
    if (this.creating()) {
      return;
    }

    const w = this.workout();
    if (!w) {
      return;
    }

    this.creating.set(true);
    this.error.set(null);

    try {
      const response = await firstValueFrom(
        this.api.createRouteSegmentFromWorkout(w.id, {
          name: this.name(),
          start: this.start(),
          end: this.end(),
        }),
      );

      if (response) {
        // Navigate to the created route segment
        this.router.navigate(['/route-segments', response.results.id]);
      }
    } catch (err) {
      console.error('Failed to create route segment:', err);
      this.error.set('Failed to create route segment. Please try again.');
      this.creating.set(false);
    }
  }

  public goBack(): void {
    const w = this.workout();
    if (w) {
      this.router.navigate(['/workouts', w.id]);
    } else {
      this.router.navigate(['/workouts']);
    }
  }
}
