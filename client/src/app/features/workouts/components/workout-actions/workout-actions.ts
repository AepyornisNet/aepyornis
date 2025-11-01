import { Component, inject, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { Api } from '../../../../core/services/api';
import { Workout, WorkoutDetail } from '../../../../core/types/workout';
import { TranslatePipe } from '@ngx-translate/core';

@Component({
  selector: 'app-workout-actions',
  imports: [CommonModule, AppIcon, TranslatePipe],
  templateUrl: './workout-actions.html',
  styleUrl: './workout-actions.scss',
})
export class WorkoutActionsComponent {
  readonly workout = input.required<Workout | WorkoutDetail>();
  readonly compact = input<boolean>(false);

  workoutUpdated = output<Workout>();
  workoutDeleted = output<void>();

  private api = inject(Api);
  private router = inject(Router);

  showDeleteConfirm = false;
  showShareMenu = false;
  isProcessing = false;
  errorMessage: string | null = null;
  successMessage: string | null = null;

  toggleLock() {
    if (this.isProcessing) {
      return;
    }

    this.isProcessing = true;
    this.errorMessage = null;

    this.api.toggleWorkoutLock(this.workout().id).subscribe({
      next: (response) => {
        this.isProcessing = false;
        this.workoutUpdated.emit(response.results);
        this.successMessage = this.workout().locked ? 'Workout unlocked' : 'Workout locked';
        setTimeout(() => (this.successMessage = null), 3000);
      },
      error: (err) => {
        this.isProcessing = false;
        this.errorMessage = 'Failed to toggle lock: ' + (err.error?.errors?.[0] || err.message);
        setTimeout(() => (this.errorMessage = null), 5000);
      },
    });
  }

  download() {
    if (this.isProcessing || !this.workout().has_file) {
      return;
    }

    this.isProcessing = true;
    this.errorMessage = null;

    this.api.downloadWorkout(this.workout().id).subscribe({
      next: (response) => {
        this.isProcessing = false;

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

        this.successMessage = 'Download started';
        setTimeout(() => (this.successMessage = null), 3000);
      },
      error: (err) => {
        this.isProcessing = false;
        this.errorMessage = 'Failed to download: ' + (err.error?.errors?.[0] || err.message);
        setTimeout(() => (this.errorMessage = null), 5000);
      },
    });
  }

  edit() {
    this.router.navigate(['/workouts', this.workout().id, 'edit']);
  }

  refresh() {
    if (this.isProcessing || !this.workout().has_file) {
      return;
    }

    this.isProcessing = true;
    this.errorMessage = null;

    this.api.refreshWorkout(this.workout().id).subscribe({
      next: (response) => {
        this.isProcessing = false;
        this.successMessage = response.results.message;
        setTimeout(() => (this.successMessage = null), 3000);
      },
      error: (err) => {
        this.isProcessing = false;
        this.errorMessage = 'Failed to refresh: ' + (err.error?.errors?.[0] || err.message);
        setTimeout(() => (this.errorMessage = null), 5000);
      },
    });
  }

  confirmDelete() {
    this.showDeleteConfirm = true;
  }

  cancelDelete() {
    this.showDeleteConfirm = false;
  }

  delete() {
    if (this.isProcessing) {
      return;
    }

    this.isProcessing = true;
    this.errorMessage = null;

    this.api.deleteWorkout(this.workout().id).subscribe({
      next: () => {
        this.isProcessing = false;
        this.showDeleteConfirm = false;
        this.workoutDeleted.emit();
        this.router.navigate(['/workouts']);
      },
      error: (err) => {
        this.isProcessing = false;
        this.errorMessage = 'Failed to delete: ' + (err.error?.errors?.[0] || err.message);
        setTimeout(() => (this.errorMessage = null), 5000);
      },
    });
  }

  toggleShareMenu() {
    this.showShareMenu = !this.showShareMenu;
  }

  generateShareLink() {
    if (this.isProcessing) {
      return;
    }

    this.isProcessing = true;
    this.errorMessage = null;

    this.api.shareWorkout(this.workout().id).subscribe({
      next: (response) => {
        this.isProcessing = false;
        this.successMessage = response.results.message;

        // Update workout with new public_uuid
        const updatedWorkout = { ...this.workout(), public_uuid: response.results.public_uuid };
        this.workoutUpdated.emit(updatedWorkout as Workout);

        setTimeout(() => (this.successMessage = null), 3000);
      },
      error: (err) => {
        this.isProcessing = false;
        this.errorMessage =
          'Failed to generate share link: ' + (err.error?.errors?.[0] || err.message);
        setTimeout(() => (this.errorMessage = null), 5000);
      },
    });
  }

  copyShareLink() {
    if (!this.workout().public_uuid) {
      return;
    }

    const shareUrl = `${window.location.origin}/share/${this.workout().public_uuid}`;
    navigator.clipboard
      .writeText(shareUrl)
      .then(() => {
        this.successMessage = 'Share link copied to clipboard';
        setTimeout(() => (this.successMessage = null), 3000);
      })
      .catch((err) => {
        this.errorMessage = 'Failed to copy to clipboard: ' + err.message;
        setTimeout(() => (this.errorMessage = null), 5000);
      });
  }

  deleteShareLink() {
    if (this.isProcessing) {
      return;
    }

    this.isProcessing = true;
    this.errorMessage = null;

    this.api.deleteWorkoutShare(this.workout().id).subscribe({
      next: (response) => {
        this.isProcessing = false;
        this.successMessage = response.results.message;

        // Update workout with removed public_uuid
        const updatedWorkout = { ...this.workout(), public_uuid: undefined };
        this.workoutUpdated.emit(updatedWorkout as Workout);

        setTimeout(() => (this.successMessage = null), 3000);
      },
      error: (err) => {
        this.isProcessing = false;
        this.errorMessage =
          'Failed to delete share link: ' + (err.error?.errors?.[0] || err.message);
        setTimeout(() => (this.errorMessage = null), 5000);
      },
    });
  }
}
