import { ChangeDetectionStrategy, Component, inject, OnInit, signal } from '@angular/core';

import { FormBuilder, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslatePipe } from '@ngx-translate/core';
import { firstValueFrom } from 'rxjs';
import { Api } from '../../../../core/services/api';
import { RouteSegmentDetail } from '../../../../core/types/route-segment';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { WORKOUT_TYPES } from '../../../../core/types/workout-types';
import { getSportLabel } from '../../../../core/i18n/sport-labels';
import { FormatDistancePipe } from '../../../../core/pipes/format-distance.pipe';
import { FormatElevationPipe } from '../../../../core/pipes/format-elevation.pipe';

@Component({
  selector: 'app-edit-route-segment',
  imports: [ReactiveFormsModule, AppIcon, TranslatePipe, FormatDistancePipe, FormatElevationPipe],
  templateUrl: './edit-route-segment.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class EditRouteSegment implements OnInit {
  public readonly sportLabel = getSportLabel;
  private api = inject(Api);
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private fb = inject(FormBuilder);

  public readonly routeSegment = signal<RouteSegmentDetail | null>(null);
  public readonly loading = signal(true);
  public readonly saving = signal(false);
  public readonly error = signal<string | null>(null);

  public readonly availableTypes = signal<string[]>([]);
  public readonly selectedCategory = signal<string>('');

  // Reactive form
  public routeSegmentForm!: FormGroup;

  public ngOnInit(): void {
    // Initialize form
    this.routeSegmentForm = this.fb.group({
      name: ['', Validators.required],
      notes: [''],
      category: [''],
      visibility: ['public', Validators.required],
      difficulty: [''],
      description: [''],
      bidirectional: [false],
      circular: [false],
    });

    this.loadFilterOptions();

    this.routeSegmentForm.get('category')?.valueChanges.subscribe((val) => {
      this.selectedCategory.set(val || '');
    });

    const id = this.route.snapshot.params['id'];
    if (id) {
      this.loadRouteSegment(parseInt(id));
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
        this.selectedCategory.set(segment.category || '');

        // Populate form with loaded data
        this.routeSegmentForm.patchValue({
          name: segment.name,
          notes: segment.notes || '',
          category: segment.category || '',
          visibility: segment.visibility || 'public',
          difficulty: segment.difficulty || '',
          description: segment.description || '',
          bidirectional: segment.bidirectional,
          circular: segment.circular,
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
    if (!segment || this.saving() || this.routeSegmentForm.invalid) {
      return;
    }

    this.saving.set(true);
    this.error.set(null);

    try {
      const formValue = this.routeSegmentForm.value;
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
      this.routeSegmentForm.patchValue({
        name: segment.name,
        notes: segment.notes || '',
        category: segment.category || '',
        visibility: segment.visibility || 'public',
        difficulty: segment.difficulty || '',
        description: segment.description || '',
        bidirectional: segment.bidirectional,
        circular: segment.circular,
      });
    }
  }
}
