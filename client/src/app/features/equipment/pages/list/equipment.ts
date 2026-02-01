import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';

import { _, TranslatePipe } from '@ngx-translate/core';
import { Router } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { Api } from '../../../../core/services/api';
import { Equipment as EquipmentModel } from '../../../../core/types/equipment';
import { PaginationParams } from '../../../../core/types/api-response';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { BaseList, BaseListConfig } from '../../../../core/components/base-list/base-list';
import { PaginatedListView } from '../../../../core/components/paginated-list-view/paginated-list-view';

@Component({
  selector: 'app-equipment',
  imports: [AppIcon, BaseList, TranslatePipe],
  templateUrl: './equipment.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Equipment extends PaginatedListView<EquipmentModel> {
  private api = inject(Api);
  private router = inject(Router);

  // Alias for better template readability
  public equipment = this.items;
  public readonly hasEquipment = computed(() => this.hasItems());

  public readonly equipmentListConfig: BaseListConfig = {
    title: _('Equipment'),
    addButtonText: _('Add equipment'),
  };

  // Modal state
  public readonly showCreateModal = signal(false);
  public readonly showEditModal = signal(false);
  public readonly showDeleteModal = signal(false);
  public readonly selectedEquipment = signal<EquipmentModel | null>(null);

  // Form state
  public readonly equipmentForm = signal({
    name: '',
    description: '',
    notes: '',
    active: true,
  });

  // Form update helpers
  public updateFormName(value: string): void {
    const form = this.equipmentForm();
    this.equipmentForm.set({ ...form, name: value });
  }

  public updateFormDescription(value: string): void {
    const form = this.equipmentForm();
    this.equipmentForm.set({ ...form, description: value });
  }

  public updateFormNotes(value: string): void {
    const form = this.equipmentForm();
    this.equipmentForm.set({ ...form, notes: value });
  }

  public updateFormActive(value: boolean): void {
    const form = this.equipmentForm();
    this.equipmentForm.set({ ...form, active: value });
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
    this.equipmentForm.set({
      name: '',
      description: '',
      notes: '',
      active: true,
    });
    this.showCreateModal.set(true);
  }
  public closeCreateModal(): void {
    this.showCreateModal.set(false);
  }

  public async createEquipment(): Promise<void> {
    try {
      const form = this.equipmentForm();
      await firstValueFrom(this.api.createEquipment(form));
      this.closeCreateModal();
      this.loadData();
    } catch (err) {
      console.error('Failed to create equipment:', err);
      this.error.set('Failed to create equipment. Please try again.');
    }
  }

  public openEditModal(equipment: EquipmentModel): void {
    this.selectedEquipment.set(equipment);
    this.equipmentForm.set({
      name: equipment.name,
      description: equipment.description || '',
      notes: equipment.notes || '',
      active: equipment.active,
    });
    this.showEditModal.set(true);
  }

  public closeEditModal(): void {
    this.showEditModal.set(false);
    this.selectedEquipment.set(null);
  }

  public async updateEquipment(): Promise<void> {
    const equipment = this.selectedEquipment();
    if (!equipment) {
      return;
    }

    try {
      const form = this.equipmentForm();
      await firstValueFrom(this.api.updateEquipment(equipment.id, form));
      this.closeEditModal();
      this.loadData();
    } catch (err) {
      console.error('Failed to update equipment:', err);
      this.error.set('Failed to update equipment. Please try again.');
    }
  }

  public openDeleteModal(equipment: EquipmentModel): void {
    this.selectedEquipment.set(equipment);
    this.showDeleteModal.set(true);
  }

  public closeDeleteModal(): void {
    this.showDeleteModal.set(false);
    this.selectedEquipment.set(null);
  }

  public async deleteEquipment(): Promise<void> {
    const equipment = this.selectedEquipment();
    if (!equipment) {
      return;
    }

    try {
      await firstValueFrom(this.api.deleteEquipment(equipment.id));
      this.closeDeleteModal();
      this.loadData();
    } catch (err) {
      console.error('Failed to delete equipment:', err);
      this.error.set('Failed to delete equipment. Please try again.');
    }
  }

  public viewDetails(equipment: EquipmentModel): void {
    this.router.navigate(['/equipment', equipment.id]);
  }
}
