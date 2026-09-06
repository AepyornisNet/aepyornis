import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  OnInit,
  signal,
} from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { form, FormField, FormRoot, max, min, required } from '@angular/forms/signals';
import { firstValueFrom } from 'rxjs';
import { Api } from '../../../../core/services/api';
import { RouteSegmentDifficulty } from '../../../../core/types/route-segment';
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

  public readonly availableTypes = signal<string[]>([]);

  // Form model & form
  public readonly routeSegmentModel = signal({
    name: '',
    category: '',
    start: 1,
    end: 1,
    difficulty: '' as RouteSegmentDifficulty,
    visibility: 'public' as 'public' | 'followers' | '' | 'private',
    description: '',
    bidirectional: false,
    circular: false,
    notes: '',
  });

  public readonly routeSegmentForm = form(
    this.routeSegmentModel,
    (s) => {
      required(s.name);
      required(s.start);
      min(s.start, 1);
      max(s.start, () => this.totalPoints() || 1);
      required(s.end);
      min(s.end, 1);
      max(s.end, () => this.totalPoints() || 1);
    },
    {
      submission: {
        action: () => this.createRouteSegment(),
      },
    },
  );

  // Computed values
  public readonly validPoints = computed(() => {
    const w = this.workout();
    const positions = w?.records?.details?.position;
    const distances = w?.records?.details?.distance;
    if (!positions || positions.length === 0) {
      return [];
    }

    const valid: { lat: number; lng: number; distance: number; originalIndex: number }[] = [];
    for (let i = 0; i < positions.length; i++) {
      const pos = positions[i];
      if (
        pos &&
        (pos[0] !== 0 || pos[1] !== 0) &&
        Number.isFinite(pos[0]) &&
        Number.isFinite(pos[1])
      ) {
        valid.push({
          lat: pos[0],
          lng: pos[1],
          distance: distances?.[i] ?? 0,
          originalIndex: i,
        });
      }
    }
    return valid;
  });

  public readonly totalPoints = computed(() => {
    return this.validPoints().length;
  });

  public readonly selectedDistance = computed(() => {
    const pts = this.validPoints();
    if (pts.length < 2) {
      return 0;
    }
    const model = this.routeSegmentModel();
    const startIdx = model.start - 1;
    const endIdx = model.end - 1;

    if (startIdx < 0 || endIdx < 0 || startIdx >= pts.length || endIdx >= pts.length) {
      return 0;
    }

    return Math.abs(pts[endIdx].distance - pts[startIdx].distance) * 1000;
  });

  public readonly workoutPoints = computed(() => {
    return this.validPoints().map((p) => ({
      lat: p.lat,
      lng: p.lng,
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
    this.loadFilterOptions();

    this.route.params.subscribe((params) => {
      const id = parseInt(params['id'], 10);
      if (id) {
        this.loadWorkout(id);
      }
    });
  }

  private async loadFilterOptions(): Promise<void> {
    const typesSet = new Set<string>();
    WORKOUT_TYPES.forEach((t) => {
      if (t.value !== 'all' && t.value !== 'auto') {
        typesSet.add(t.value);
      }
    });

    try {
      const res = await firstValueFrom(this.api.getWorkoutFilterOptions());
      if (res?.results?.types?.length) {
        res.results.types.forEach((t) => typesSet.add(t));
      }
    } catch (err) {
      console.error('Failed to load filter options:', err);
    }

    this.availableTypes.set(Array.from(typesSet));
  }

  public async loadWorkout(id: number): Promise<void> {
    this.loading.set(true);
    this.error.set(null);

    try {
      const response = await firstValueFrom(this.api.getWorkout(id));

      if (response) {
        const workout = response.results;
        this.workout.set(workout);

        // Set end to the last valid point
        const points = this.validPoints().length || 1;
        this.routeSegmentModel.set({
          name: workout.name || '',
          category: workout.type || '',
          start: 1,
          end: points,
          difficulty: '',
          visibility: (workout.visibility || 'public') as 'public' | 'followers' | '' | 'private',
          description: '',
          bidirectional: false,
          circular: false,
          notes: '',
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
    const total = this.totalPoints();
    const clamped = Math.max(1, Math.min(value, total || 1));
    this.routeSegmentModel.update((m) => ({
      ...m,
      start: clamped,
      end: clamped > m.end ? clamped : m.end,
    }));
  }

  public updateEnd(value: number): void {
    const total = this.totalPoints();
    const clamped = Math.max(1, Math.min(value, total || 1));
    this.routeSegmentModel.update((m) => ({
      ...m,
      end: clamped,
      start: clamped < m.start ? clamped : m.start,
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
          difficulty: formValue.difficulty || undefined,
          visibility: formValue.visibility || undefined,
          description: formValue.description || undefined,
          notes: formValue.notes || undefined,
          bidirectional: formValue.bidirectional,
          circular: formValue.circular,
        }),
      );
      const created = response?.results;

      if (!created) {
        this.error.set(this.translate.instant('Failed to create route segment. Please try again.'));
        return;
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
