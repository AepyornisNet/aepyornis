import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  signal,
  viewChild,
} from '@angular/core';

import { RouterLink } from '@angular/router';
import { firstValueFrom, Observable } from 'rxjs';
import { Api } from '../../../../core/services/api';
import { Workout } from '../../../../core/types/workout';
import { WorkoutListParams } from '../../../../core/types/workout';
import { WORKOUT_TYPES } from '../../../../core/types/workout-types';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { PaginatedListView } from '../../../../core/components/paginated-list-view/paginated-list-view';
import { WorkoutListActions } from '../../components/workout-list-actions/workout-list-actions';
import { TranslatePipe, TranslateService, Translation } from '@ngx-translate/core';
import { BaseList, BaseListConfig } from '../../../../core/components/base-list/base-list';
import { BaseTable } from '../../../../core/components/base-table/base-table';
import { AsyncPipe } from '@angular/common';

type WorkoutListFilterState = {
  type: string;
  since: string;
  orderBy: string;
  orderDir: 'desc' | 'asc';
};

type FilterOption = {
  value: string;
  label: Observable<Translation>;
};

@Component({
  selector: 'app-workouts',
  imports: [RouterLink, AppIcon, WorkoutListActions, TranslatePipe, BaseList, BaseTable, AsyncPipe],
  templateUrl: './workouts.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Workouts extends PaginatedListView<Workout> {
  private api = inject(Api);
  private translate = inject(TranslateService);

  public readonly baseList = viewChild.required(BaseList);

  // Alias for better template readability
  public workouts = this.items;
  public readonly hasWorkouts = computed(() => this.hasItems());

  public readonly listConfig: BaseListConfig = {
    title: 'menu.workouts',
    addButtonText: 'workout.add_workout',
    enableSearch: false,
    enableFilters: true,
    enableMultiSelect: true,
  };

  public readonly workoutTypes = WORKOUT_TYPES;

  public readonly sinceOptions: FilterOption[] = [
    { value: 'forever', label: this.translate.stream('filters.forever') },
    { value: '7 days', label: this.translate.stream('filters.n_days', { num : 7 }) },
    { value: '15 days', label: this.translate.stream('filters.n_days', { num: 15 }) },
    { value: '1 month', label: this.translate.stream('filters.1_month') },
    { value: '3 months', label: this.translate.stream('filters.n_months', { num: 3 }) },
    { value: '6 months', label: this.translate.stream('filters.n_months', { num: 6 }) },
    { value: '1 year', label: this.translate.stream('filters.1_year') },
    { value: '2 years', label: this.translate.stream('filters.n_years', { num: 2 }) },
    { value: '5 years', label: this.translate.stream('filters.n_years', { num: 5 }) },
    { value: '10 years', label: this.translate.stream('filters.n_years', { num: 10 }) },
  ];

  public readonly orderByOptions: FilterOption[] = [
    { value: 'date', label: this.translate.stream('misc.date') },
    { value: 'total_distance', label: this.translate.stream('workout.distance') },
    { value: 'total_duration', label: this.translate.stream('workout.duration') },
    { value: 'total_weight', label: this.translate.stream('workout.weight') },
    { value: 'total_repetitions', label: this.translate.stream('workout.repetitions') },
    { value: 'total_up', label: this.translate.stream('workout.elev_up') },
    { value: 'total_down', label: this.translate.stream('workout.elev_down') },
    { value: 'average_speed_no_pause', label: this.translate.stream('workout.average_speed_no_pause') },
    { value: 'max_speed', label: this.translate.stream('workout.max_speed') },
  ];

  public readonly orderDirOptions: FilterOption[] = [
    { value: 'desc', label: this.translate.stream('filters.descending') },
    { value: 'asc', label: this.translate.stream('filters.ascending') },
  ];

  private readonly _filters = signal<WorkoutListFilterState>({
    type: '',
    since: '10 years',
    orderBy: 'date',
    orderDir: 'desc',
  });
  public readonly filterState = computed(() => this._filters());

  private readonly _orderByWithOwnColumn = new Set(['date', 'total_distance', 'total_duration', 'average_speed']);
  public readonly sortedColumnMeta = computed<FilterOption | null>(() => {
    const orderBy = this.filterState().orderBy;
    if (this._orderByWithOwnColumn.has(orderBy)) {
      return null;
    }
    return this.orderByOptions.find((option) => option.value === orderBy) ?? null;
  });

  public readonly getWorkoutLink = (workout: Workout): (string | number)[] => ['/workouts', workout.id];

  public getSortedColumnValue(workout: Workout): string {
    const orderBy = this.filterState().orderBy;
    switch (orderBy) {
      case 'total_weight': {
        const formatted = this.formatNumber(workout.total_weight, { maximumFractionDigits: 1 });
        return formatted !== null ? formatted : '-';
      }
      case 'total_repetitions': {
        const formatted = this.formatNumber(workout.total_repetitions, { maximumFractionDigits: 0 });
        return formatted !== null ? formatted : '-';
      }
      case 'total_up': {
        const formatted = this.formatNumber(workout.total_up, { maximumFractionDigits: 0 });
        return formatted !== null ? `${formatted} m` : '-';
      }
      case 'total_down': {
        const formatted = this.formatNumber(workout.total_down, { maximumFractionDigits: 0 });
        return formatted !== null ? `${formatted} m` : '-';
      }
      case 'average_speed_no_pause': {
        const speed = this.formatSpeed(workout.average_speed_no_pause);
        return speed !== null ? `${speed} km/h` : '-';
      }
      case 'max_speed': {
        const speed = this.formatSpeed(workout.max_speed);
        return speed !== null ? `${speed} km/h` : '-';
      }
      default:
        return '-';
    }
  }

  public async loadData(page?: number): Promise<void> {
    if (page) {
      this.currentPage.set(page);
    }

    this.loading.set(true);
    this.error.set(null);

    const filters = this.filterState();

    const params: WorkoutListParams = {
      page: this.currentPage(),
      per_page: this.perPage(),
      since: filters.since,
      order_by: filters.orderBy,
      order_dir: filters.orderDir,
    };

    if (filters.type) {
      params.type = filters.type;
    }

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

  public onAddWorkout(): void {
    window.location.href = '/workouts/add';
  }

  public formatDate(dateString: string): string {
    return new Date(dateString).toLocaleDateString();
  }

  public formatDistance(distance: number): string {
    return (distance / 1000).toFixed(2);
  }

  public formatDuration(seconds: number): string {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    if (hours > 0) {
      return `${hours}h ${minutes}m`;
    }
    return `${minutes}m`;
  }

  public onWorkoutUpdated(workout: Workout): void {
    const index = this.items().findIndex((w) => w.id === workout.id);
    if (index >= 0) {
      const updatedItems = [...this.items()];
      updatedItems[index] = { ...updatedItems[index], ...workout };
      this.items.set(updatedItems);
    }
  }

  public onWorkoutDeleted(workoutId: number): void {
    const updatedItems = this.items().filter((w) => w.id !== workoutId);
    this.items.set(updatedItems);
    this.totalCount.update((count) => count - 1);

    // Remove from selection if it was selected
    const baseListComponent = this.baseList();
    if (baseListComponent.isItemSelected(workoutId)) {
      baseListComponent.toggleItemSelection(workoutId);
    }
  }

  public bulkDelete(): void {
    const selectedIds = Array.from(this.baseList().selectedItems());
    if (selectedIds.length === 0 || !confirm(`Delete ${selectedIds.length} workouts?`)) {
      return;
    }

    // Delete all selected workouts
    Promise.all(
      selectedIds.map((id) => firstValueFrom(this.api.deleteWorkout(id as number))),
    ).then(
      () => {
        // Reload data after deletion
        this.baseList().clearSelection();
        this.loadData(this.currentPage());
      },
      (err) => {
        console.error('Failed to delete workouts:', err);
        this.error.set('Failed to delete some workouts. Please try again.');
      },
    );
  }

  public isMultiSelectActive(): boolean {
    return this.baseList().multiSelectActive();
  }

  public isItemSelected(id: number): boolean {
    return this.baseList().isItemSelected(id);
  }

  public toggleItemSelection(id: number | string): void {
    this.baseList().toggleItemSelection(id);
  }

  private handleFilterChange(update: Partial<WorkoutListFilterState>): void {
    this._filters.update((state) => ({
      ...state,
      ...update,
    }));
    this.loadData(1);
  }

  public onWorkoutTypeFilterChange(value: string): void {
    this.handleFilterChange({ type: value });
  }

  public onWorkoutSinceFilterChange(value: string): void {
    this.handleFilterChange({ since: value });
  }

  public onWorkoutOrderByChange(value: string): void {
    this.handleFilterChange({ orderBy: value });
  }

  public onWorkoutOrderDirChange(value: string): void {
    const dir: WorkoutListFilterState['orderDir'] = value === 'asc' ? 'asc' : 'desc';
    this.handleFilterChange({ orderDir: dir });
  }

  private formatNumber(value: number | null | undefined, options?: Intl.NumberFormatOptions): string | null {
    if (value === undefined || value === null) {
      return null;
    }
    return value.toLocaleString(undefined, options);
  }

  private formatSpeed(value: number | null | undefined): string | null {
    if (value === undefined || value === null) {
      return null;
    }
    return (value * 3.6).toFixed(2);
  }
}
