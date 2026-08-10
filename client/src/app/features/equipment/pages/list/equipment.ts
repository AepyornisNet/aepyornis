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
import { Router, RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { NgbModal, NgbModalRef } from '@ng-bootstrap/ng-bootstrap';
import { Api } from '../../../../core/services/api';
import { Equipment as EquipmentModel } from '../../../../core/types/equipment';
import { PaginationParams } from '../../../../core/types/api-response';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { BaseList, BaseListConfig } from '../../../../core/components/base-list/base-list';
import { PaginatedListView } from '../../../../core/components/paginated-list-view/paginated-list-view';
import { BaseTable } from '../../../../core/components/base-table/base-table';
import { WORKOUT_TYPES } from '../../../../core/types/workout-types';

import { EquipmentForm, EquipmentFormData } from '../../components/equipment-form/equipment-form';

@Component({
  selector: 'app-equipment',
  imports: [AppIcon, BaseList, BaseTable, TranslatePipe, RouterLink, EquipmentForm],
  templateUrl: './equipment.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Equipment extends PaginatedListView<EquipmentModel> {
  private api = inject(Api);
  private router = inject(Router);
  private modalService = inject(NgbModal);

  public readonly createModalTemplate = viewChild<TemplateRef<unknown>>('createModal');
  public readonly editModalTemplate = viewChild<TemplateRef<unknown>>('editModal');
  public readonly deleteModalTemplate = viewChild<TemplateRef<unknown>>('deleteModal');
  private activeModalRef?: NgbModalRef;

  // Alias for better template readability
  public equipment = this.items;
  public readonly hasEquipment = computed(() => this.hasItems());

  public readonly equipmentListConfig: BaseListConfig = {
    title: _('Equipment'),
    addButtonText: _('Add equipment'),
  };

  public readonly workoutTypes = WORKOUT_TYPES;

  public readonly selectedEquipment = signal<EquipmentModel | null>(null);

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
      const response = await firstValueFrom(this.api.getEquipment(params));

      if (response) {
        this.updatePaginationState(response);
      }
    } catch (err) {
      console.error('Failed to load equipment:', err);
      this.error.set('Failed to load equipment. Please try again.');
    } finally {
      this.loading.set(false);
    }
  }

  public formatDate(dateString: string): string {
    return new Date(dateString).toLocaleDateString();
  }

  public openCreateModal(): void {
    this.selectedEquipment.set(null);
    const template = this.createModalTemplate();
    if (template) {
      this.activeModalRef = this.modalService.open(template, { centered: true });
    }
  }

  public closeCreateModal(): void {
    this.activeModalRef?.dismiss();
  }

  public async createEquipment(formData: EquipmentFormData): Promise<void> {
    try {
      await firstValueFrom(this.api.createEquipment(formData));
      this.activeModalRef?.close();
      this.loadData();
    } catch (err) {
      console.error('Failed to create equipment:', err);
      this.error.set('Failed to create equipment. Please try again.');
    }
  }

  public openEditModal(equipment: EquipmentModel): void {
    this.selectedEquipment.set(equipment);
    const template = this.editModalTemplate();
    if (template) {
      this.activeModalRef = this.modalService.open(template, { centered: true });
    }
  }

  public closeEditModal(): void {
    this.activeModalRef?.dismiss();
    this.selectedEquipment.set(null);
  }

  public async updateEquipment(formData: EquipmentFormData): Promise<void> {
    const equipment = this.selectedEquipment();
    if (!equipment) {
      return;
    }

    try {
      await firstValueFrom(this.api.updateEquipment(equipment.id, formData));
      this.activeModalRef?.close();
      this.selectedEquipment.set(null);
      this.loadData();
    } catch (err) {
      console.error('Failed to update equipment:', err);
      this.error.set('Failed to update equipment. Please try again.');
    }
  }

  public openDeleteModal(equipment: EquipmentModel): void {
    this.selectedEquipment.set(equipment);
    const template = this.deleteModalTemplate();
    if (template) {
      this.activeModalRef = this.modalService.open(template, { centered: true });
    }
  }

  public closeDeleteModal(): void {
    this.activeModalRef?.dismiss();
    this.selectedEquipment.set(null);
  }

  public async deleteEquipment(): Promise<void> {
    const equipment = this.selectedEquipment();
    if (!equipment) {
      return;
    }

    try {
      await firstValueFrom(this.api.deleteEquipment(equipment.id));
      this.activeModalRef?.close();
      this.selectedEquipment.set(null);
      this.loadData();
    } catch (err) {
      console.error('Failed to delete equipment:', err);
      this.error.set('Failed to delete equipment. Please try again.');
    }
  }

  public viewDetails(equipment: EquipmentModel): void {
    this.router.navigate(['/equipment', equipment.id]);
  }

  public readonly getEquipmentLink = (equipment: EquipmentModel): (string | number)[] => [
    '/equipment',
    equipment.id,
  ];
}
