import { Component, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { TranslatePipe } from '@ngx-translate/core';
import { firstValueFrom } from 'rxjs';
import { Api } from '../../../../core/services/api';
import { Measurement } from '../../../../core/types/measurement';
import { PaginationParams } from '../../../../core/types/api-response';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { PaginatedListView } from '../../../../core/components/paginated-list-view/paginated-list-view';

@Component({
  selector: 'app-measurements',
  imports: [CommonModule, AppIcon, TranslatePipe],
  templateUrl: './measurements.html',
})
export class Measurements extends PaginatedListView<Measurement> {
  private api = inject(Api);

  // Alias for better template readability
  measurements = this.items;
  readonly hasMeasurements = computed(() => this.hasItems());

  // Modal state
  readonly showCreateModal = signal(false);
  readonly showEditModal = signal(false);
  readonly showDeleteModal = signal(false);
  readonly selectedMeasurement = signal<Measurement | null>(null);

  // Form state
  readonly measurementForm = signal({
    date: '',
    weight: null as number | null,
    height: null as number | null,
    steps: null as number | null,
  });

  // Form update helpers
  updateFormDate(value: string) {
    const form = this.measurementForm();
    this.measurementForm.set({ ...form, date: value });
  }

  updateFormWeight(value: string) {
    const form = this.measurementForm();
    this.measurementForm.set({ ...form, weight: value ? parseFloat(value) : null });
  }

  updateFormHeight(value: string) {
    const form = this.measurementForm();
    this.measurementForm.set({ ...form, height: value ? parseFloat(value) : null });
  }

  updateFormSteps(value: string) {
    const form = this.measurementForm();
    this.measurementForm.set({ ...form, steps: value ? parseInt(value) : null });
  }

  async loadData(page?: number) {
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

  formatDate(dateString: string): string {
    return new Date(dateString).toLocaleDateString();
  }

  formatDateForInput(dateString: string): string {
    const date = new Date(dateString);
    return date.toISOString().split('T')[0];
  }

  getTodayDate(): string {
    return new Date().toISOString().split('T')[0];
  }

  openCreateModal() {
    this.measurementForm.set({
      date: this.getTodayDate(),
      weight: null,
      height: null,
      steps: null,
    });
    this.showCreateModal.set(true);
  }

  closeCreateModal() {
    this.showCreateModal.set(false);
  }

  async createMeasurement() {
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

  openEditModal(measurement: Measurement) {
    this.selectedMeasurement.set(measurement);
    this.measurementForm.set({
      date: this.formatDateForInput(measurement.date),
      weight: measurement.weight || null,
      height: measurement.height || null,
      steps: measurement.steps || null,
    });
    this.showEditModal.set(true);
  }

  closeEditModal() {
    this.showEditModal.set(false);
    this.selectedMeasurement.set(null);
  }

  async updateMeasurement() {
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

  openDeleteModal(measurement: Measurement) {
    this.selectedMeasurement.set(measurement);
    this.showDeleteModal.set(true);
  }

  closeDeleteModal() {
    this.showDeleteModal.set(false);
    this.selectedMeasurement.set(null);
  }

  async deleteMeasurement() {
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
