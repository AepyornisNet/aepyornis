import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  OnInit,
  signal,
} from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { form, FormField, FormRoot, min, required } from '@angular/forms/signals';
import { firstValueFrom } from 'rxjs';
import { Api } from '../../../../core/services/api';
import { WorkoutDetail } from '../../../../core/types/workout';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { TranslatePipe, TranslateService } from '@ngx-translate/core';
import { RouteSegmentMapComponent } from '../../components/route-segment-map/route-segment-map';
import { FormatDistancePipe } from '../../../../core/pipes/format-distance.pipe';
import { getSportLabel } from '../../../../core/i18n/sport-labels';
import { WORKOUT_TYPES } from '../../../../core/types/workout-types';

@Component({
  selector: 'app-create-workout-route-segment',
  imports: [
    FormField,
    FormRoot,
    AppIcon,
    TranslatePipe,
    RouteSegmentMapComponent,
    FormatDistancePipe,
  ],
  templateUrl: './create-workout-route-segment.html',
  styleUrl: './create-workout-route-segment.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class CreateWorkoutRouteSegmentPage implements OnInit {
  public readonly sportLabel = getSportLabel;
  private api = inject(Api);
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private translate = inject(TranslateService);

  public readonly workout = signal<WorkoutDetail | null>(null);
  public readonly loading = signal(true);
  public readonly error = signal<string | null>(null);
  public readonly creating = signal(false);

  public readonly availableTypes = signal<string[]>(WORKOUT_TYPES.map((t) => t.value));

  // Form model & form
  public readonly routeSegmentModel = signal({
    name: '',
    category: '',
    start: 1,
    end: 1,
    bidirectional: false,
    circular: false,
  });

  public readonly routeSegmentForm = form(
    this.routeSegmentModel,
    (s) => {
      required(s.name);
      min(s.start, 1);
      min(s.end, 1);
    },
    {
      submission: {
        action: () => this.createRouteSegment(),
      },
    },
  );

  // Computed values
  public readonly totalPoints = computed(() => {
    const w = this.workout();
    return w?.records?.details?.position?.length || 0;
  });

  public readonly selectedDistance = computed(() => {
    const w = this.workout();
    const model = this.routeSegmentModel();
    const startIdx = model.start - 1;
    const endIdx = model.end - 1;

    if (!w?.records?.details?.distance || startIdx < 0 || endIdx < 0) {
      return 0;
    }

    const distances = w.records?.details.distance;
    if (endIdx >= distances.length || startIdx >= distances.length) {
      return 0;
    }

    return Math.abs(distances[endIdx] - distances[startIdx]); // convert to km
  });

  public readonly workoutPoints = computed(() => {
    const w = this.workout();
    if (!w?.records?.details?.position) {
      return [];
    }
    return w.records.details.position.map((p: [number, number]) => ({
      lat: p[0],
      lng: p[1],
    }));
  });

  public readonly selection = computed(() => {
    const total = this.totalPoints();
    if (total < 2) {
      return null;
    }
    const model = this.routeSegmentModel();
    const startIdx = Math.max(0, Math.min(model.start - 1, total - 2));
    const endIdx = Math.max(startIdx + 1, Math.min(model.end - 1, total - 1));
    return { startIndex: startIdx, endIndex: endIdx };
  });

  public ngOnInit(): void {
    this.route.params.subscribe((params) => {
      const id = parseInt(params['id'], 10);
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

        // Set end to the last point
        const points = workout.records?.details?.position?.length || 1;
        this.routeSegmentModel.set({
          name: workout.name || '',
          category: workout.type || '',
          start: 1,
          end: points,
          bidirectional: false,
          circular: false,
        });
      }
    } catch (err) {
      console.error('Failed to load workout:', err);
      this.error.set(this.translate.instant('Failed to load workout. Please try again.'));
    } finally {
      this.loading.set(false);
    }
  }

  public updateStart(value: number): void {
    this.routeSegmentModel.update((m) => ({
      ...m,
      start: value,
      end: value > m.end ? value : m.end,
    }));
  }

  public updateEnd(value: number): void {
    this.routeSegmentModel.update((m) => ({
      ...m,
      end: value,
      start: value < m.start ? value : m.start,
    }));
  }

  public async createRouteSegment(): Promise<void> {
    if (this.creating() || this.routeSegmentForm().invalid()) {
      return;
    }
    const w = this.workout();
    if (!w) {
      return;
    }

    this.creating.set(true);
    this.error.set(null);

    try {
      const formValue = this.routeSegmentModel();
      const response = await firstValueFrom(
        this.api.createRouteSegmentFromWorkout(w.id, {
          name: formValue.name,
          start: formValue.start,
          end: formValue.end,
          category: formValue.category || undefined,
        }),
      );
      const created = response?.results;

      if (!created) {
        this.error.set(this.translate.instant('Failed to create route segment. Please try again.'));
        return;
      }

      if (formValue.bidirectional || formValue.circular) {
        await firstValueFrom(
          this.api.updateRouteSegment(created.id, {
            name: created.name ?? formValue.name,
            notes: created.notes ?? '',
            bidirectional: formValue.bidirectional,
            circular: formValue.circular,
          }),
        );
      }

      this.router.navigate(['/route-segments', created.id]);
    } catch (err) {
      console.error('Failed to create route segment:', err);
      this.error.set(this.translate.instant('Failed to create route segment. Please try again.'));
    } finally {
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
