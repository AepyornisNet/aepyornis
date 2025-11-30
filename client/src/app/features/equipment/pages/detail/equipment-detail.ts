import { ChangeDetectionStrategy, Component, inject, OnInit, signal } from '@angular/core';

import { ActivatedRoute, Router } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { Api } from '../../../../core/services/api';
import { Equipment } from '../../../../core/types/equipment';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { TranslatePipe } from '@ngx-translate/core';

@Component({
  selector: 'app-equipment-detail',
  imports: [AppIcon, TranslatePipe],
  templateUrl: './equipment-detail.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class EquipmentDetail implements OnInit {
  private api = inject(Api);
  private route = inject(ActivatedRoute);
  private router = inject(Router);

  public readonly equipment = signal<Equipment | null>(null);
  public readonly loading = signal(true);
  public readonly error = signal<string | null>(null);

  // Modal state
  public readonly showEditModal = signal(false);
  public readonly showDeleteModal = signal(false);

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

  public ngOnInit(): void {
    this.route.params.subscribe((params) => {
      const id = parseInt(params['id']);
      if (id) {
        this.loadEquipment(id);
      }
    });
  }

  public async loadEquipment(id: number): Promise<void> {
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

  public formatDate(dateString: string): string {
    return new Date(dateString).toLocaleDateString();
  }

  public openEditModal(): void {
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

  public closeEditModal(): void {
    this.showEditModal.set(false);
  }

  public async updateEquipment(): Promise<void> {
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

  public openDeleteModal(): void {
    this.showDeleteModal.set(true);
  }

  public closeDeleteModal(): void {
    this.showDeleteModal.set(false);
  }

  public async deleteEquipment(): Promise<void> {
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

  public goBack(): void {
    this.router.navigate(['/equipment']);
  }
}
