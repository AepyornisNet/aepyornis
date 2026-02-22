import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  OnInit,
  signal,
} from '@angular/core';
import { HttpErrorResponse } from '@angular/common/http';

import { FormBuilder, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { Api } from '../../../../core/services/api';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { Equipment } from '../../../../core/types/equipment';
import {
  getWorkoutTypeConfig,
  WORKOUT_TYPES,
  WorkoutTypeConfig,
} from '../../../../core/types/workout-types';
import { TranslatePipe, TranslateService } from '@ngx-translate/core';

@Component({
  selector: 'app-workout-create',
  imports: [ReactiveFormsModule, AppIcon, TranslatePipe],
  templateUrl: './workout-create.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class WorkoutCreate implements OnInit {
  private api = inject(Api);
  private router = inject(Router);
  private route = inject(ActivatedRoute);
  private fb = inject(FormBuilder);
  private translate = inject(TranslateService);

  // Edit mode
  public readonly editMode = signal(false);
  public readonly workoutId = signal<number | null>(null);

  // State
  public readonly loading = signal(false);
  public readonly error = signal<string | null>(null);
  public readonly success = signal<string | null>(null);

  // Equipment list
  public readonly equipment = signal<Equipment[]>([]);

  // File upload form
  public readonly selectedFiles = signal<File[]>([]);
  public fileUploadForm!: FormGroup;

  // Manual form
  public manualWorkoutForm!: FormGroup;
  private readonly _manualWorkoutType = signal<string>('');
  public readonly manualWorkoutType = computed(() => this._manualWorkoutType());
  public readonly manualFormVisible = computed(() => this._manualWorkoutType() !== '');

  // Computed properties for conditional field display
  public readonly workoutTypeConfig = computed<WorkoutTypeConfig | undefined>(() => {
    const type = this.manualWorkoutType();
    return type ? getWorkoutTypeConfig(type) : undefined;
  });

  public readonly showLocation = computed(() => this.workoutTypeConfig()?.location ?? false);
  public readonly showDistance = computed(() => this.workoutTypeConfig()?.distance ?? false);
  public readonly showDuration = computed(() => this.workoutTypeConfig()?.duration ?? false);
  public readonly showRepetitions = computed(() => this.workoutTypeConfig()?.repetition ?? false);
  public readonly showWeight = computed(() => this.workoutTypeConfig()?.weight ?? false);
  public readonly showCustomType = computed(() => this.manualWorkoutType() === 'other');

  // Available workout types
  public readonly workoutTypes = WORKOUT_TYPES;

  public ngOnInit(): void {
    // Initialize file upload form
    this.fileUploadForm = this.fb.group({
      type: ['auto'],
      notes: [''],
    });

    // Initialize manual workout form
    this.manualWorkoutForm = this.fb.group({
      name: ['', Validators.required],
      date: [this.getDefaultDateTime(), Validators.required],
      visibility: [''],
      location: [''],
      duration_hours: [0, [Validators.required, Validators.min(0)]],
      duration_minutes: [0, [Validators.required, Validators.min(0), Validators.max(59)]],
      duration_seconds: [0, [Validators.required, Validators.min(0), Validators.max(59)]],
      distance: [0, [Validators.required, Validators.min(0)]],
      repetitions: [0, [Validators.required, Validators.min(0)]],
      weight: [0, [Validators.required, Validators.min(0)]],
      notes: [''],
      custom_type: [''],
      equipment_ids: [[]],
    });

    // Check if we're in edit mode
    const id = this.route.snapshot.paramMap.get('id');
    if (id) {
      this.editMode.set(true);
      this.workoutId.set(parseInt(id, 10));
      this.loadWorkoutForEdit(parseInt(id, 10));
    } else {
      this.loadDefaultWorkoutVisibility();
    }
    this.loadEquipment();
  }

  private async loadDefaultWorkoutVisibility(): Promise<void> {
    try {
      const profileResponse = await firstValueFrom(this.api.getProfile());
      const defaultVisibility = profileResponse?.results?.profile?.default_workout_visibility ?? '';
      this.manualWorkoutForm.patchValue({ visibility: defaultVisibility });
    } catch (err) {
      console.error('Failed to load default workout visibility:', err);
    }
  }

  public async loadWorkoutForEdit(id: number): Promise<void> {
    this.loading.set(true);
    this.error.set(null);

    try {
      const response = await firstValueFrom(this.api.getWorkout(id));

      if (response && response.results) {
        const workout = response.results;

        // Set manual workout type
        this._manualWorkoutType.set(workout.type);

        // Parse date to local datetime format
        const date = new Date(workout.date);
        const year = date.getFullYear();
        const month = String(date.getMonth() + 1).padStart(2, '0');
        const day = String(date.getDate()).padStart(2, '0');
        const hours = String(date.getHours()).padStart(2, '0');
        const minutes = String(date.getMinutes()).padStart(2, '0');
        const formattedDate = `${year}-${month}-${day}T${hours}:${minutes}`;

        // Calculate duration components from total_duration (in seconds)
        const totalSeconds = workout.total_duration || 0;
        const durationHours = Math.floor(totalSeconds / 3600);
        const durationMinutes = Math.floor((totalSeconds % 3600) / 60);
        const durationSeconds = totalSeconds % 60;

        // Update form with workout data
        this.manualWorkoutForm.patchValue({
          name: workout.name,
          date: formattedDate,
          visibility: workout.visibility ?? '',
          location: workout.address_string || '',
          duration_hours: durationHours,
          duration_minutes: durationMinutes,
          duration_seconds: durationSeconds,
          distance: workout.total_distance ? workout.total_distance / 1000 : 0, // Convert meters to km
          repetitions: workout.total_repetitions || 0,
          weight: workout.total_weight || 0,
          notes: workout.notes || '',
          custom_type: workout.custom_type || '',
          equipment_ids: workout.equipment?.map((e) => e.id) || [],
        });
      }
    } catch (err) {
      console.error('Failed to load workout:', err);
      this.error.set(this.translate.instant('Failed to load workout. Please try again.'));
    } finally {
      this.loading.set(false);
    }
  }

  private getDefaultDateTime(): string {
    const now = new Date();
    const year = now.getFullYear();
    const month = String(now.getMonth() + 1).padStart(2, '0');
    const day = String(now.getDate()).padStart(2, '0');
    const hours = String(now.getHours()).padStart(2, '0');
    const minutes = String(now.getMinutes()).padStart(2, '0');
    return `${year}-${month}-${day}T${hours}:${minutes}`;
  }

  private getTimezone(): string {
    return Intl.DateTimeFormat().resolvedOptions().timeZone;
  }

  public async loadEquipment(): Promise<void> {
    try {
      const response = await firstValueFrom(this.api.getEquipment({ page: 1, per_page: 100 }));
      if (response) {
        this.equipment.set(response.results);
      }
    } catch (err) {
      console.error('Failed to load equipment:', err);
    }
  }

  // File upload handlers
  public onFilesSelected(event: Event): void {
    const input = event.target as HTMLInputElement;
    if (input.files) {
      this.selectedFiles.set(Array.from(input.files));
    }
  }

  public removeFile(index: number): void {
    const files = this.selectedFiles();
    files.splice(index, 1);
    this.selectedFiles.set([...files]);
  }

  public async submitFileUpload(): Promise<void> {
    const files = this.selectedFiles();
    if (files.length === 0) {
      this.error.set('Please select at least one file');
      return;
    }

    this.loading.set(true);
    this.error.set(null);
    this.success.set(null);

    try {
      const formValue = this.fileUploadForm.value;
      const formData = new FormData();
      files.forEach((file) => {
        formData.append('file', file);
      });
      // Send empty string for autodetect, otherwise send the selected type
      const uploadType = formValue.type === 'auto' ? '' : formValue.type;
      formData.append('type', uploadType);
      formData.append('notes', formValue.notes);

      const response = await firstValueFrom(this.api.createWorkoutFromFile(formData));

      if (response) {
        this.success.set(`Successfully created ${response.results.length} workout(s)`);
        // Reset form
        this.selectedFiles.set([]);
        this.fileUploadForm.reset({ type: 'auto', notes: '' });
        // Navigate to workouts page after a short delay
        setTimeout(() => {
          this.router.navigate(['/workouts']);
        }, 1500);
      }
    } catch (err) {
      console.error('Failed to upload workouts:', err);
      const apiError = this.extractApiError(err);
      this.error.set(
        apiError || this.translate.instant('Failed to upload workouts. Please try again.'),
      );
    } finally {
      this.loading.set(false);
    }
  }

  private extractApiError(err: unknown): string | null {
    if (err instanceof HttpErrorResponse) {
      const apiErrorCodes = err.error?.error_codes;
      if (Array.isArray(apiErrorCodes) && apiErrorCodes.length > 0) {
        const mapped = this.mapApiErrorCode(apiErrorCodes[0]);
        if (mapped) {
          return mapped;
        }
      }

      const apiErrors = err.error?.errors;
      if (Array.isArray(apiErrors) && apiErrors.length > 0) {
        return apiErrors[0];
      }

      if (typeof err.error === 'string' && err.error.length > 0) {
        return err.error;
      }

      if (typeof err.message === 'string' && err.message.length > 0) {
        return err.message;
      }
    }

    return null;
  }

  private mapApiErrorCode(code: string): string | null {
    switch (code) {
      case 'workout_already_exists':
        return this.translate.instant('A workout with the same start time already exists.');
      default:
        return null;
    }
  }
  // Manual form handlers
  public updateManualWorkoutType(value: string): void {
    this._manualWorkoutType.set(value);
    // Pre-fill name with workout type and timestamp
    if (value) {
      const now = new Date();
      const timestamp = now.toISOString();
      const displayName = value.replace(/-/g, ' ');
      this.manualWorkoutForm.patchValue({ name: `${displayName} - ${timestamp}` });
    }
  }

  public toggleEquipment(equipmentId: number): void {
    const currentIds = this.manualWorkoutForm.value.equipment_ids || [];
    const index = currentIds.indexOf(equipmentId);
    if (index > -1) {
      currentIds.splice(index, 1);
    } else {
      currentIds.push(equipmentId);
    }
    this.manualWorkoutForm.patchValue({ equipment_ids: [...currentIds] });
  }

  public isEquipmentSelected(equipmentId: number): boolean {
    const ids = this.manualWorkoutForm.value.equipment_ids || [];
    return ids.includes(equipmentId);
  }

  public async submitManualWorkout(): Promise<void> {
    const type = this._manualWorkoutType();

    if (!type) {
      this.error.set('Please select a workout type');
      return;
    }

    if (this.manualWorkoutForm.invalid) {
      this.error.set('Please fill in all required fields');
      return;
    }

    this.loading.set(true);
    this.error.set(null);
    this.success.set(null);

    try {
      const formValue = this.manualWorkoutForm.value;
      const workoutData: {
        name: string;
        date: string;
        timezone: string;
        type: string;
        visibility: '' | 'followers' | 'public';
        notes: string;
        equipment_ids: number[];
        location?: string;
        duration_hours?: number;
        duration_minutes?: number;
        duration_seconds?: number;
        distance?: number;
        repetitions?: number;
        weight?: number;
        custom_type?: string;
      } = {
        name: formValue.name,
        date: formValue.date,
        timezone: this.getTimezone(),
        type,
        visibility: formValue.visibility,
        notes: formValue.notes,
        equipment_ids: formValue.equipment_ids,
      };

      if (this.showLocation()) {
        workoutData.location = formValue.location;
      }

      if (this.showDuration()) {
        workoutData.duration_hours = formValue.duration_hours;
        workoutData.duration_minutes = formValue.duration_minutes;
        workoutData.duration_seconds = formValue.duration_seconds;
      }

      if (this.showDistance()) {
        workoutData.distance = formValue.distance;
      }

      if (this.showRepetitions()) {
        workoutData.repetitions = formValue.repetitions;
      }

      if (this.showWeight()) {
        workoutData.weight = formValue.weight;
      }

      if (this.showCustomType()) {
        workoutData.custom_type = formValue.custom_type;
      }

      let response;
      if (this.editMode()) {
        // Update existing workout
        response = await firstValueFrom(this.api.updateWorkout(this.workoutId()!, workoutData));
      } else {
        // Create new workout
        response = await firstValueFrom(this.api.createWorkoutManual(workoutData));
      }

      if (response) {
        this.success.set(
          this.editMode() ? 'Workout updated successfully' : 'Workout created successfully',
        );
        // Navigate to workout detail after a short delay
        setTimeout(() => {
          this.router.navigate(['/workouts', response.results.id]);
        }, 1500);
      }
    } catch (err) {
      console.error(`Failed to ${this.editMode() ? 'update' : 'create'} workout:`, err);
      const apiError = this.extractApiError(err);
      const fallbackError = this.editMode()
        ? this.translate.instant('Failed to update workout. Please try again.')
        : this.translate.instant('Failed to create workout. Please try again.');

      this.error.set(
        apiError || fallbackError,
      );
    } finally {
      this.loading.set(false);
    }
  }

  public navigateToWorkouts(): void {
    this.router.navigate(['/workouts']);
  }
}
