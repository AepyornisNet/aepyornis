import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';

import { TranslatePipe } from '@ngx-translate/core';
import { firstValueFrom } from 'rxjs';
import { Api } from '../../../../core/services/api';
import { Measurement } from '../../../../core/types/measurement';
import { PaginationParams } from '../../../../core/types/api-response';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { PaginatedListView } from '../../../../core/components/paginated-list-view/paginated-list-view';

@Component({
  selector: 'app-measurements',
  imports: [AppIcon, TranslatePipe],
  templateUrl: './measurements.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Measurements extends PaginatedListView<Measurement> {
  private api = inject(Api);

  // Alias for better template readability
  public measurements = this.items;
  public readonly hasMeasurements = computed(() => this.hasItems());

  // Modal state
  public readonly showCreateModal = signal(false);
  public readonly showEditModal = signal(false);
  public readonly showDeleteModal = signal(false);
  public readonly selectedMeasurement = signal<Measurement | null>(null);

  // Form state
  public readonly measurementForm = signal({
    date: '',
    weight: null as number | null,
    height: null as number | null,
    steps: null as number | null,
  });

  // Form update helpers
  public updateFormDate(value: string): void {
    const form = this.measurementForm();
    this.measurementForm.set({ ...form, date: value });
  }

  public updateFormWeight(value: string): void {
    const form = this.measurementForm();
    this.measurementForm.set({ ...form, weight: value ? parseFloat(value) : null });
  }

  public updateFormHeight(value: string): void {
    const form = this.measurementForm();
    this.measurementForm.set({ ...form, height: value ? parseFloat(value) : null });
  }

  public updateFormSteps(value: string): void {
    const form = this.measurementForm();
    this.measurementForm.set({ ...form, steps: value ? parseInt(value) : null });
  }

  public async loadData(page?: number): Promise<void> {
    if (page) {
      this.currentPage.set(page);
    }

    this.loading.set(true);
    this.error.set(null);

    const params: PaginationParams = {
      page: this.currentPage(),
      per_page: this.perPage(),
    };

    try {
      const response = await firstValueFrom(this.api.getMeasurements(params));

      if (response) {
        this.updatePaginationState(response);
      }
    } catch (err) {
      console.error('Failed to load measurements:', err);
      this.error.set('Failed to load measurements. Please try again.');
    } finally {
      this.loading.set(false);
    }
  }

  public formatDate(dateString: string): string {
    return new Date(dateString).toLocaleDateString();
  }

  public formatDateForInput(dateString: string): string {
    const date = new Date(dateString);
    return date.toISOString().split('T')[0];
  }

  public getTodayDate(): string {
    return new Date().toISOString().split('T')[0];
  }

  public openCreateModal(): void {
    this.measurementForm.set({
      date: this.getTodayDate(),
      weight: null,
      height: null,
      steps: null,
    });
    this.showCreateModal.set(true);
  }

  public closeCreateModal(): void {
    this.showCreateModal.set(false);
  }

  public async createMeasurement(): Promise<void> {
    try {
      const form = this.measurementForm();
      if (!form.date) {
        this.error.set('Date is required');
        return;
      }

      const payload: {
        date: string;
        weight?: number;
        height?: number;
        steps?: number;
      } = { date: form.date };
      if (form.weight !== null && form.weight > 0) {
        payload.weight = form.weight;
      }
      if (form.height !== null && form.height > 0) {
        payload.height = form.height;
      }
      if (form.steps !== null && form.steps > 0) {
        payload.steps = form.steps;
      }

      await firstValueFrom(this.api.createOrUpdateMeasurement(payload));
      this.closeCreateModal();
      this.loadData();
    } catch (err) {
      console.error('Failed to create measurement:', err);
      this.error.set('Failed to create measurement. Please try again.');
    }
  }

  public openEditModal(measurement: Measurement): void {
    this.selectedMeasurement.set(measurement);
    this.measurementForm.set({
      date: this.formatDateForInput(measurement.date),
      weight: measurement.weight || null,
      height: measurement.height || null,
      steps: measurement.steps || null,
    });
    this.showEditModal.set(true);
  }

  public closeEditModal(): void {
    this.showEditModal.set(false);
    this.selectedMeasurement.set(null);
  }

  public async updateMeasurement(): Promise<void> {
    const measurement = this.selectedMeasurement();
    if (!measurement) {
      return;
    }

    try {
      const form = this.measurementForm();
      const payload: {
        date: string;
        weight?: number;
        height?: number;
        steps?: number;
      } = { date: form.date };
      if (form.weight !== null && form.weight > 0) {
        payload.weight = form.weight;
      }
      if (form.height !== null && form.height > 0) {
        payload.height = form.height;
      }
      if (form.steps !== null && form.steps > 0) {
        payload.steps = form.steps;
      }

      await firstValueFrom(this.api.createOrUpdateMeasurement(payload));
      this.closeEditModal();
      this.loadData();
    } catch (err) {
      console.error('Failed to update measurement:', err);
      this.error.set('Failed to update measurement. Please try again.');
    }
  }

  public openDeleteModal(measurement: Measurement): void {
    this.selectedMeasurement.set(measurement);
    this.showDeleteModal.set(true);
  }

  public closeDeleteModal(): void {
    this.showDeleteModal.set(false);
    this.selectedMeasurement.set(null);
  }

  public async deleteMeasurement(): Promise<void> {
    const measurement = this.selectedMeasurement();
    if (!measurement) {
      return;
    }

    try {
      const dateStr = this.formatDateForInput(measurement.date);
      await firstValueFrom(this.api.deleteMeasurement(dateStr));
      this.closeDeleteModal();
      this.loadData();
    } catch (err) {
      console.error('Failed to delete measurement:', err);
      this.error.set('Failed to delete measurement. Please try again.');
    }
  }
}
