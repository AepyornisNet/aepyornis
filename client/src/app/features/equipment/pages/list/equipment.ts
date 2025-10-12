import { Component, signal, computed, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { Api } from '../../../../core/services/api';
import { Equipment as EquipmentModel } from '../../../../core/types/equipment';
import { PaginationParams } from '../../../../core/types/api-response';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { PaginatedListView } from '../../../../core/components/paginated-list-view/paginated-list-view';

@Component({
  selector: 'app-equipment',
  imports: [CommonModule, AppIcon],
  templateUrl: './equipment.html'
})
export class Equipment extends PaginatedListView<EquipmentModel> {
  private api = inject(Api);
  private router = inject(Router);

  // Alias for better template readability
  equipment = this.items;
  hasEquipment = computed(() => this.hasItems());

  // Modal state
  showCreateModal = signal(false);
  showEditModal = signal(false);
  showDeleteModal = signal(false);
  selectedEquipment = signal<EquipmentModel | null>(null);

  // Form state
  equipmentForm = signal({
    name: '',
    description: '',
    notes: '',
    active: true
  });



  // Form update helpers
  updateFormName(value: string) {
    const form = this.equipmentForm();
    this.equipmentForm.set({ ...form, name: value });
  }

  updateFormDescription(value: string) {
    const form = this.equipmentForm();
    this.equipmentForm.set({ ...form, description: value });
  }

  updateFormNotes(value: string) {
    const form = this.equipmentForm();
    this.equipmentForm.set({ ...form, notes: value });
  }

  updateFormActive(value: boolean) {
    const form = this.equipmentForm();
    this.equipmentForm.set({ ...form, active: value });
  }

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

  formatDate(dateString: string): string {
    return new Date(dateString).toLocaleDateString();
  }

  openCreateModal() {
    this.equipmentForm.set({
      name: '',
      description: '',
      notes: '',
      active: true
    });
    this.showCreateModal.set(true);
  }

  closeCreateModal() {
    this.showCreateModal.set(false);
  }

  async createEquipment() {
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

  openEditModal(equipment: EquipmentModel) {
    this.selectedEquipment.set(equipment);
    this.equipmentForm.set({
      name: equipment.name,
      description: equipment.description || '',
      notes: equipment.notes || '',
      active: equipment.active
    });
    this.showEditModal.set(true);
  }

  closeEditModal() {
    this.showEditModal.set(false);
    this.selectedEquipment.set(null);
  }

  async updateEquipment() {
    const equipment = this.selectedEquipment();
    if (!equipment) return;

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

  openDeleteModal(equipment: EquipmentModel) {
    this.selectedEquipment.set(equipment);
    this.showDeleteModal.set(true);
  }

  closeDeleteModal() {
    this.showDeleteModal.set(false);
    this.selectedEquipment.set(null);
  }

  async deleteEquipment() {
    const equipment = this.selectedEquipment();
    if (!equipment) return;

    try {
      await firstValueFrom(this.api.deleteEquipment(equipment.id));
      this.closeDeleteModal();
      this.loadData();
    } catch (err) {
      console.error('Failed to delete equipment:', err);
      this.error.set('Failed to delete equipment. Please try again.');
    }
  }

  viewDetails(equipment: EquipmentModel) {
    this.router.navigate(['/equipment', equipment.id]);
  }
}
