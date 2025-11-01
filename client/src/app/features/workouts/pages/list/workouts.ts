import { Component, inject, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { Api } from '../../../../core/services/api';
import { Workout } from '../../../../core/types/workout';
import { PaginationParams } from '../../../../core/types/api-response';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { PaginatedListView } from '../../../../core/components/paginated-list-view/paginated-list-view';
import { WorkoutActionsComponent } from '../../components/workout-actions/workout-actions';
import { Pagination } from '../../../../core/components/pagination/pagination';

// TODO: Implement filtering and sorting of workouts
@Component({
  selector: 'app-workouts',
  imports: [CommonModule, RouterLink, AppIcon, WorkoutActionsComponent, Pagination],
  templateUrl: './workouts.html'
})
export class Workouts extends PaginatedListView<Workout> {
  private api = inject(Api);

  // Alias for better template readability
  workouts = this.items;
  hasWorkouts = computed(() => this.hasItems());

  async loadData(page?: number) {
    if (page) {
      this.currentPage.set(page);
    }

    this.loading.set(true);
    this.error.set(null);

    const params: PaginationParams = {
      page: this.currentPage(),
      per_page: this.perPage()
    };

    try {
      const response = await firstValueFrom(this.api.getWorkouts(params));

      if (response) {
        this.updatePaginationState(response);
      }
    } catch (err) {
      console.error('Failed to load workouts:', err);
      this.error.set('Failed to load workouts. Please try again.');
    } finally {
      this.loading.set(false);
    }
  }

  formatDate(dateString: string): string {
    return new Date(dateString).toLocaleDateString();
  }

  formatDistance(distance: number): string {
    return (distance / 1000).toFixed(2);
  }

  formatDuration(seconds: number): string {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    if (hours > 0) {
      return `${hours}h ${minutes}m`;
    }
    return `${minutes}m`;
  }

  onWorkoutUpdated(workout: Workout) {
    // Update the workout in the list
    const index = this.items().findIndex(w => w.id === workout.id);
    if (index >= 0) {
      const updatedItems = [...this.items()];
      updatedItems[index] = { ...updatedItems[index], ...workout };
      this.items.set(updatedItems);
    }
  }

  onWorkoutDeleted(workoutId: number) {
    // Remove workout from the list
    const updatedItems = this.items().filter(w => w.id !== workoutId);
    this.items.set(updatedItems);
    this.totalCount.update(count => count - 1);
  }
}
