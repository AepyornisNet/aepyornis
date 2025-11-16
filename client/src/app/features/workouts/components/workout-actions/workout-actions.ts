import { ChangeDetectionStrategy, Component, computed, inject, input, output, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router, RouterLink } from '@angular/router';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { Api } from '../../../../core/services/api';
import { Workout, WorkoutDetail } from '../../../../core/types/workout';
import { TranslatePipe } from '@ngx-translate/core';
import { NgbDropdownModule } from '@ng-bootstrap/ng-bootstrap';
import { User } from '../../../../core/services/user';

@Component({
  selector: 'app-workout-actions',
  imports: [CommonModule, AppIcon, TranslatePipe, NgbDropdownModule, RouterLink],
  templateUrl: './workout-actions.html',
  styleUrl: './workout-actions.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class WorkoutActions {
  public readonly workout = input.required<Workout | WorkoutDetail>();
  public readonly compact = input<boolean>(false);
  public readonly hasMapData = input<boolean>(false);

  public readonly workoutUpdated = output<Workout>();
  public readonly workoutDeleted = output<void>();

  private api = inject(Api);
  private router = inject(Router);
  private userService = inject(User);

  public readonly showDeleteConfirm = signal(false);
  public readonly showShareMenu = signal(false);
  public readonly isProcessing = signal(false);
  public readonly errorMessage = signal<string | null>(null);
  public readonly successMessage = signal<string | null>(null);

  // Check if socials are disabled from user profile
  public readonly socialsDisabled = computed(() => {
    const userInfo = this.userService.getUserInfo()();
    return userInfo?.profile?.socials_disabled ?? false;
  });

  public toggleLock(): void {
    if (this.isProcessing()) {
      return;
    }

    this.isProcessing.set(true);
    this.errorMessage.set(null);

    this.api.toggleWorkoutLock(this.workout().id).subscribe({
      next: (response) => {
        this.isProcessing.set(false);
        this.workoutUpdated.emit(response.results);
        const message = response.results.locked ? 'Workout locked' : 'Workout unlocked';
        this.successMessage.set(message);
        setTimeout(() => this.successMessage.set(null), 3000);
      },
      error: (err) => {
        this.isProcessing.set(false);
        this.errorMessage.set('Failed to toggle lock: ' + (err.error?.errors?.[0] || err.message));
        setTimeout(() => this.errorMessage.set(null), 5000);
      },
    });
  }

  public download(): void {
    if (this.isProcessing() || !this.workout().has_file) {
      return;
    }

    this.isProcessing.set(true);
    this.errorMessage.set(null);

    this.api.downloadWorkout(this.workout().id).subscribe({
      next: (response) => {
        this.isProcessing.set(false);

        // Create download link
        if (response.body) {
          const url = window.URL.createObjectURL(response.body);
          const a = document.createElement('a');
          a.href = url;
          const contentDisposition = response.headers.get('content-disposition');
          a.download = contentDisposition
            ? contentDisposition.split('filename=')[1]?.replace(/"/g, '')
            : 'workout.gpx';
          document.body.appendChild(a);
          a.click();
          window.URL.revokeObjectURL(url);
          document.body.removeChild(a);
        }

        this.successMessage.set('Download started');
        setTimeout(() => this.successMessage.set(null), 3000);
      },
      error: (err) => {
        this.isProcessing.set(false);
        this.errorMessage.set('Failed to download: ' + (err.error?.errors?.[0] || err.message));
        setTimeout(() => this.errorMessage.set(null), 5000);
      },
    });
  }

  public edit(): void {
    this.router.navigate(['/workouts', this.workout().id, 'edit']);
  }

  public refresh(): void {
    if (this.isProcessing() || !this.workout().has_file) {
      return;
    }

    this.isProcessing.set(true);
    this.errorMessage.set(null);

    this.api.refreshWorkout(this.workout().id).subscribe({
      next: (response) => {
        this.isProcessing.set(false);
        this.successMessage.set(response.results.message);
        setTimeout(() => this.successMessage.set(null), 3000);
      },
      error: (err) => {
        this.isProcessing.set(false);
        this.errorMessage.set('Failed to refresh: ' + (err.error?.errors?.[0] || err.message));
        setTimeout(() => this.errorMessage.set(null), 5000);
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
    this.errorMessage.set(null);

    this.api.deleteWorkout(this.workout().id).subscribe({
      next: () => {
        this.isProcessing.set(false);
        this.showDeleteConfirm.set(false);
        this.workoutDeleted.emit();
        this.router.navigate(['/workouts']);
      },
      error: (err) => {
        this.isProcessing.set(false);
        this.errorMessage.set('Failed to delete: ' + (err.error?.errors?.[0] || err.message));
        setTimeout(() => this.errorMessage.set(null), 5000);
      },
    });
  }

  public toggleShareMenu(): void {
    this.showShareMenu.update((value) => !value);
  }

  public generateShareLink(): void {
    if (this.isProcessing()) {
      return;
    }

    this.closeShareMenu();
    this.isProcessing.set(true);
    this.errorMessage.set(null);

    this.api.shareWorkout(this.workout().id).subscribe({
      next: (response) => {
        this.isProcessing.set(false);
        this.successMessage.set(response.results.message);

        // Update workout with new public_uuid
        const updatedWorkout = { ...this.workout(), public_uuid: response.results.public_uuid };
        this.workoutUpdated.emit(updatedWorkout as Workout);

        setTimeout(() => this.successMessage.set(null), 3000);
      },
      error: (err) => {
        this.isProcessing.set(false);
        this.errorMessage.set(
          'Failed to generate share link: ' + (err.error?.errors?.[0] || err.message),
        );
        setTimeout(() => this.errorMessage.set(null), 5000);
      },
    });
  }

  public copyShareLink(): void {
    if (!this.workout().public_uuid) {
      return;
    }

    this.closeShareMenu();
    const shareUrl = `${window.location.origin}/share/${this.workout().public_uuid}`;
    navigator.clipboard
      .writeText(shareUrl)
      .then(() => {
        this.successMessage.set('Share link copied to clipboard');
        setTimeout(() => this.successMessage.set(null), 3000);
      })
      .catch((err) => {
        this.errorMessage.set('Failed to copy to clipboard: ' + err.message);
        setTimeout(() => this.errorMessage.set(null), 5000);
      });
  }

  public deleteShareLink(): void {
    if (this.isProcessing()) {
      return;
    }

    this.closeShareMenu();
    this.isProcessing.set(true);
    this.errorMessage.set(null);

    this.api.deleteWorkoutShare(this.workout().id).subscribe({
      next: (response) => {
        this.isProcessing.set(false);
        this.successMessage.set(response.results.message);

        // Update workout with removed public_uuid
        const updatedWorkout = { ...this.workout(), public_uuid: undefined };
        this.workoutUpdated.emit(updatedWorkout as Workout);

        setTimeout(() => this.successMessage.set(null), 3000);
      },
      error: (err) => {
        this.isProcessing.set(false);
        this.errorMessage.set(
          'Failed to delete share link: ' + (err.error?.errors?.[0] || err.message),
        );
        setTimeout(() => this.errorMessage.set(null), 5000);
      },
    });
  }

  private closeShareMenu(): void {
    this.showShareMenu.set(false);
  }
}
