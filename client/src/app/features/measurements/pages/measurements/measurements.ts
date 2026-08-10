import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  signal,
  TemplateRef,
  viewChild,
} from '@angular/core';

import { _, TranslatePipe } from '@ngx-translate/core';
import { firstValueFrom } from 'rxjs';
import { NgbModal, NgbModalRef } from '@ng-bootstrap/ng-bootstrap';
import { Api } from '../../../../core/services/api';
import { Measurement } from '../../../../core/types/measurement';
import { PaginationParams } from '../../../../core/types/api-response';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { BaseList, BaseListConfig } from '../../../../core/components/base-list/base-list';
import { BaseTable } from '../../../../core/components/base-table/base-table';
import { PaginatedListView } from '../../../../core/components/paginated-list-view/paginated-list-view';

import { FormatDatePipe } from '../../../../core/pipes/format-date.pipe';

import {
  MeasurementForm,
  MeasurementFormData,
} from '../../components/measurement-form/measurement-form';

@Component({
  selector: 'app-measurements',
  imports: [AppIcon, BaseList, BaseTable, TranslatePipe, FormatDatePipe, MeasurementForm],
  templateUrl: './measurements.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Measurements extends PaginatedListView<Measurement> {
  private api = inject(Api);
  private modalService = inject(NgbModal);

  public readonly createModalTemplate = viewChild<TemplateRef<unknown>>('createModal');
  public readonly editModalTemplate = viewChild<TemplateRef<unknown>>('editModal');
  public readonly deleteModalTemplate = viewChild<TemplateRef<unknown>>('deleteModal');
  private activeModalRef?: NgbModalRef;

  public readonly measurementListConfig: BaseListConfig = {
    title: _('Measurements'),
    addButtonText: _('Add measurement'),
  };

  // Alias for better template readability
  public measurements = this.items;
  public readonly hasMeasurements = computed(() => this.hasItems());

  public readonly selectedMeasurement = signal<Measurement | null>(null);

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

  public openCreateModal(): void {
    this.selectedMeasurement.set(null);
    const template = this.createModalTemplate();
    if (template) {
      this.activeModalRef = this.modalService.open(template, { centered: true });
    }
  }

  public closeCreateModal(): void {
    this.activeModalRef?.dismiss();
  }

  public async createMeasurement(formData: MeasurementFormData): Promise<void> {
    try {
      if (!formData.date) {
        this.error.set('Date is required');
        return;
      }

      const payload: {
        date: string;
        weight?: number;
        height?: number;
        steps?: number;
        ftp?: number;
        resting_heart_rate?: number;
        max_heart_rate?: number;
      } = { date: formData.date };
      if (formData.weight !== null && formData.weight > 0) {
        payload.weight = formData.weight;
      }
      if (formData.height !== null && formData.height > 0) {
        payload.height = formData.height;
      }
      if (formData.steps !== null && formData.steps > 0) {
        payload.steps = formData.steps;
      }
      if (formData.ftp !== null && formData.ftp > 0) {
        payload.ftp = formData.ftp;
      }
      if (formData.resting_heart_rate !== null && formData.resting_heart_rate > 0) {
        payload.resting_heart_rate = formData.resting_heart_rate;
      }
      if (formData.max_heart_rate !== null && formData.max_heart_rate > 0) {
        payload.max_heart_rate = formData.max_heart_rate;
      }

      await firstValueFrom(this.api.createOrUpdateMeasurement(payload));
      this.activeModalRef?.close();
      this.loadData();
    } catch (err) {
      console.error('Failed to create measurement:', err);
      this.error.set('Failed to create measurement. Please try again.');
    }
  }

  public openEditModal(measurement: Measurement): void {
    this.selectedMeasurement.set(measurement);
    const template = this.editModalTemplate();
    if (template) {
      this.activeModalRef = this.modalService.open(template, { centered: true });
    }
  }

  public closeEditModal(): void {
    this.activeModalRef?.dismiss();
    this.selectedMeasurement.set(null);
  }

  public async updateMeasurement(formData: MeasurementFormData): Promise<void> {
    const measurement = this.selectedMeasurement();
    if (!measurement) {
      return;
    }

    try {
      const payload: {
        date: string;
        weight?: number;
        height?: number;
        steps?: number;
        ftp?: number;
        resting_heart_rate?: number;
        max_heart_rate?: number;
      } = { date: formData.date };
      if (formData.weight !== null && formData.weight > 0) {
        payload.weight = formData.weight;
      }
      if (formData.height !== null && formData.height > 0) {
        payload.height = formData.height;
      }
      if (formData.steps !== null && formData.steps > 0) {
        payload.steps = formData.steps;
      }
      if (formData.ftp !== null && formData.ftp > 0) {
        payload.ftp = formData.ftp;
      }
      if (formData.resting_heart_rate !== null && formData.resting_heart_rate > 0) {
        payload.resting_heart_rate = formData.resting_heart_rate;
      }
      if (formData.max_heart_rate !== null && formData.max_heart_rate > 0) {
        payload.max_heart_rate = formData.max_heart_rate;
      }

      await firstValueFrom(this.api.createOrUpdateMeasurement(payload));
      this.activeModalRef?.close();
      this.selectedMeasurement.set(null);
      this.loadData();
    } catch (err) {
      console.error('Failed to update measurement:', err);
      this.error.set('Failed to update measurement. Please try again.');
    }
  }

  public openDeleteModal(measurement: Measurement): void {
    this.selectedMeasurement.set(measurement);
    const template = this.deleteModalTemplate();
    if (template) {
      this.activeModalRef = this.modalService.open(template, { centered: true });
    }
  }

  public closeDeleteModal(): void {
    this.activeModalRef?.dismiss();
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
      this.activeModalRef?.close();
      this.selectedMeasurement.set(null);
      this.loadData();
    } catch (err) {
      console.error('Failed to delete measurement:', err);
      this.error.set('Failed to delete measurement. Please try again.');
    }
  }
}
