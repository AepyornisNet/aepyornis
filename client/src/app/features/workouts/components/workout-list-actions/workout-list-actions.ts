import { ChangeDetectionStrategy, Component, inject, input, output, signal } from '@angular/core';
import { Router } from '@angular/router';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { Api } from '../../../../core/services/api';
import { Workout } from '../../../../core/types/workout';
import { TranslatePipe } from '@ngx-translate/core';
import { NgbDropdownModule } from '@ng-bootstrap/ng-bootstrap';

@Component({
  selector: 'app-workout-list-actions',
  imports: [AppIcon, TranslatePipe, NgbDropdownModule],
  templateUrl: './workout-list-actions.html',
  styleUrl: './workout-list-actions.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class WorkoutListActions {
  public readonly workout = input.required<Workout>();

  public readonly workoutUpdated = output<Workout>();
  public readonly workoutDeleted = output<void>();

  private api = inject(Api);
  private router = inject(Router);

  public readonly isProcessing = signal(false);
  public readonly showDeleteConfirm = signal(false);

  public edit(): void {
    this.router.navigate(['/workouts', this.workout().id, 'edit']);
  }

  public view(): void {
    this.router.navigate(['/workouts', this.workout().id]);
  }

  public toggleLock(): void {
    if (this.isProcessing()) {
      return;
    }

    this.isProcessing.set(true);

    this.api.toggleWorkoutLock(this.workout().id).subscribe({
      next: (response) => {
        this.isProcessing.set(false);
        this.workoutUpdated.emit(response.results);
      },
      error: () => {
        this.isProcessing.set(false);
      },
    });
  }

  public confirmDelete(): void {
    this.showDeleteConfirm.set(true);
  }

  public cancelDelete(): void {
    this.showDeleteConfirm.set(false);
  }

  public delete(): void {
    if (this.isProcessing()) {
      return;
    }

    this.isProcessing.set(true);

    this.api.deleteWorkout(this.workout().id).subscribe({
      next: () => {
        this.isProcessing.set(false);
        this.showDeleteConfirm.set(false);
        this.workoutDeleted.emit();
      },
      error: () => {
        this.isProcessing.set(false);
      },
    });
  }
}
