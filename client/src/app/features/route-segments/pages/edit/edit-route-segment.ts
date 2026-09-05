import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  OnInit,
  signal,
} from '@angular/core';
import { disabled, form, FormField, FormRoot, required } from '@angular/forms/signals';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslatePipe } from '@ngx-translate/core';
import { firstValueFrom } from 'rxjs';
import { Api } from '../../../../core/services/api';
import { RouteSegmentDetail, RouteSegmentDifficulty } from '../../../../core/types/route-segment';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { WORKOUT_TYPES } from '../../../../core/types/workout-types';
import { getSportLabel } from '../../../../core/i18n/sport-labels';
import { FormatDistancePipe } from '../../../../core/pipes/format-distance.pipe';
import { FormatElevationPipe } from '../../../../core/pipes/format-elevation.pipe';

@Component({
  selector: 'app-edit-route-segment',
  imports: [FormField, FormRoot, AppIcon, TranslatePipe, FormatDistancePipe, FormatElevationPipe],
  templateUrl: './edit-route-segment.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class EditRouteSegment implements OnInit {
  public readonly sportLabel = getSportLabel;
  private api = inject(Api);
  private route = inject(ActivatedRoute);
  private router = inject(Router);

  public readonly routeSegment = signal<RouteSegmentDetail | null>(null);
  public readonly loading = signal(true);
  public readonly saving = signal(false);
  public readonly error = signal<string | null>(null);

  public readonly availableTypes = signal<string[]>([]);

  public readonly routeSegmentModel = signal({
    name: '',
    notes: '',
    category: '',
    visibility: 'public' as 'public' | 'followers' | '' | 'private',
    difficulty: '' as RouteSegmentDifficulty,
    description: '',
    bidirectional: false,
    circular: false,
  });

  public readonly selectedCategory = computed(() => this.routeSegmentModel().category);

  public readonly routeSegmentForm = form(
    this.routeSegmentModel,
    (s) => {
      required(s.name);
      required(s.visibility);
      disabled(s.name, { when: () => this.saving() });
      disabled(s.notes, { when: () => this.saving() });
      disabled(s.category, { when: () => this.saving() });
      disabled(s.visibility, { when: () => this.saving() });
      disabled(s.difficulty, { when: () => this.saving() });
      disabled(s.description, { when: () => this.saving() });
      disabled(s.bidirectional, { when: () => this.saving() });
      disabled(s.circular, { when: () => this.saving() });
    },
    {
      submission: {
        action: () => this.save(),
      },
    },
  );

  public ngOnInit(): void {
    this.loadFilterOptions();

    const id = this.route.snapshot.params['id'];
    if (id) {
      this.loadRouteSegment(parseInt(id, 10));
    }
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

  public async loadRouteSegment(id: number): Promise<void> {
    this.loading.set(true);
    this.error.set(null);

    try {
      const response = await firstValueFrom(this.api.getRouteSegment(id));

      if (response) {
        const segment = response.results;
        this.routeSegment.set(segment);

        // Populate form with loaded data
        this.routeSegmentModel.set({
          name: segment.name || '',
          notes: segment.notes || '',
          category: segment.category || '',
          visibility: (segment.visibility || 'public') as 'public' | 'followers' | '' | 'private',
          difficulty: (segment.difficulty || '') as RouteSegmentDifficulty,
          description: segment.description || '',
          bidirectional: Boolean(segment.bidirectional),
          circular: Boolean(segment.circular),
        });
      }
    } catch (err) {
      console.error('Failed to load route segment:', err);
      this.error.set('Failed to load route segment. Please try again.');
    } finally {
      this.loading.set(false);
    }
  }

  public async save(): Promise<void> {
    const segment = this.routeSegment();
    if (!segment || this.saving() || this.routeSegmentForm().invalid()) {
      return;
    }

    this.saving.set(true);
    this.error.set(null);

    try {
      const formValue = this.routeSegmentModel();
      await firstValueFrom(
        this.api.updateRouteSegment(segment.id, {
          name: formValue.name,
          notes: formValue.notes,
          category: formValue.category,
          visibility: formValue.visibility,
          difficulty: formValue.difficulty,
          description: formValue.description,
          bidirectional: formValue.bidirectional,
          circular: formValue.circular,
        }),
      );

      // Navigate back to detail page
      this.router.navigate(['/route-segments', segment.id]);
    } catch (err) {
      console.error('Failed to update route segment:', err);
      this.error.set('Failed to update route segment. Please try again.');
      this.saving.set(false);
    }
  }

  public cancel(): void {
    const segment = this.routeSegment();
    if (segment) {
      this.router.navigate(['/route-segments', segment.id]);
    } else {
      this.router.navigate(['/route-segments']);
    }
  }

  public reset(): void {
    const segment = this.routeSegment();
    if (segment) {
      this.routeSegmentModel.set({
        name: segment.name || '',
        notes: segment.notes || '',
        category: segment.category || '',
        visibility: (segment.visibility || 'public') as 'public' | 'followers' | '' | 'private',
        difficulty: (segment.difficulty || '') as RouteSegmentDifficulty,
        description: segment.description || '',
        bidirectional: Boolean(segment.bidirectional),
        circular: Boolean(segment.circular),
      });
    }
  }
}
