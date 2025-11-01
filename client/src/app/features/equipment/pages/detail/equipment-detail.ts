import { Component, inject, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { Api } from '../../../../core/services/api';
import { Equipment } from '../../../../core/types/equipment';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { TranslatePipe } from '@ngx-translate/core';

@Component({
  selector: 'app-equipment-detail',
  imports: [CommonModule, AppIcon, TranslatePipe],
  templateUrl: './equipment-detail.html',
})
export class EquipmentDetail implements OnInit {
  private api = inject(Api);
  private route = inject(ActivatedRoute);
  private router = inject(Router);

  readonly equipment = signal<Equipment | null>(null);
  readonly loading = signal(true);
  readonly error = signal<string | null>(null);

  // Modal state
  readonly showEditModal = signal(false);
  readonly showDeleteModal = signal(false);

  // Form state
  readonly equipmentForm = signal({
    name: '',
    description: '',
    notes: '',
    active: true,
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

  ngOnInit() {
    this.route.params.subscribe((params) => {
      const id = parseInt(params['id']);
      if (id) {
        this.loadEquipment(id);
      }
    });
  }

  async loadEquipment(id: number) {
    this.loading.set(true);
    this.error.set(null);

    try {
      const response = await firstValueFrom(this.api.getEquipmentById(id));

      if (response) {
        this.equipment.set(response.results);
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

  openEditModal() {
    const eq = this.equipment();
    if (!eq) {
      return;
    }

    this.equipmentForm.set({
      name: eq.name,
      description: eq.description || '',
      notes: eq.notes || '',
      active: eq.active,
    });
    this.showEditModal.set(true);
  }

  closeEditModal() {
    this.showEditModal.set(false);
  }

  async updateEquipment() {
    const eq = this.equipment();
    if (!eq) {
      return;
    }

    try {
      const form = this.equipmentForm();
      await firstValueFrom(this.api.updateEquipment(eq.id, form));
      this.closeEditModal();
      this.loadEquipment(eq.id);
    } catch (err) {
      console.error('Failed to update equipment:', err);
      this.error.set('Failed to update equipment. Please try again.');
    }
  }

  openDeleteModal() {
    this.showDeleteModal.set(true);
  }

  closeDeleteModal() {
    this.showDeleteModal.set(false);
  }

  async deleteEquipment() {
    const eq = this.equipment();
    if (!eq) {
      return;
    }

    try {
      await firstValueFrom(this.api.deleteEquipment(eq.id));
      this.router.navigate(['/equipment']);
    } catch (err) {
      console.error('Failed to delete equipment:', err);
      this.error.set('Failed to delete equipment. Please try again.');
      this.closeDeleteModal();
    }
  }

  goBack() {
    this.router.navigate(['/equipment']);
  }
}
