import {
  ChangeDetectionStrategy,
  Component,
  inject,
  OnInit,
  signal,
  TemplateRef,
  viewChild,
} from '@angular/core';

import { ActivatedRoute, Router } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { NgbModal, NgbModalRef } from '@ng-bootstrap/ng-bootstrap';
import { Api } from '../../../../core/services/api';
import { Equipment } from '../../../../core/types/equipment';
import { TranslatePipe } from '@ngx-translate/core';
import { WORKOUT_TYPES } from '../../../../core/types/workout-types';
import { getSportLabel } from '../../../../core/i18n/sport-labels';

import { EquipmentForm, EquipmentFormData } from '../../components/equipment-form/equipment-form';

@Component({
  selector: 'app-equipment-detail',
  imports: [TranslatePipe, EquipmentForm],
  templateUrl: './equipment-detail.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class EquipmentDetail implements OnInit {
  private api = inject(Api);
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private modalService = inject(NgbModal);

  public readonly editModalTemplate = viewChild<TemplateRef<unknown>>('editModal');
  public readonly deleteModalTemplate = viewChild<TemplateRef<unknown>>('deleteModal');
  private activeModalRef?: NgbModalRef;

  public readonly equipment = signal<Equipment | null>(null);
  public readonly loading = signal(true);
  public readonly error = signal<string | null>(null);

  public readonly workoutTypes = WORKOUT_TYPES;
  public readonly sportLabel = getSportLabel;

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

  public formatDistance(distance: number): string {
    return (distance / 1000).toFixed(2);
  }

  public formatDuration(seconds: number): string {
    const totalSeconds = Math.floor(seconds);
    const hours = Math.floor(totalSeconds / 3600);
    const minutes = Math.floor((totalSeconds % 3600) / 60);
    const remainingSeconds = totalSeconds % 60;

    if (hours > 0) {
      return `${hours}:${minutes.toString().padStart(2, '0')}:${remainingSeconds
        .toString()
        .padStart(2, '0')}`;
    }

    return `${minutes}:${remainingSeconds.toString().padStart(2, '0')}`;
  }

  public openEditModal(): void {
    const eq = this.equipment();
    if (!eq) {
      return;
    }

    const template = this.editModalTemplate();
    if (template) {
      this.activeModalRef = this.modalService.open(template, { centered: true });
    }
  }

  public closeEditModal(): void {
    this.activeModalRef?.dismiss();
  }

  public async updateEquipment(formData: EquipmentFormData): Promise<void> {
    const eq = this.equipment();
    if (!eq) {
      return;
    }

    try {
      await firstValueFrom(this.api.updateEquipment(eq.id, formData));
      this.activeModalRef?.close();
      this.loadEquipment(eq.id);
    } catch (err) {
      console.error('Failed to update equipment:', err);
      this.error.set('Failed to update equipment. Please try again.');
    }
  }

  public openDeleteModal(): void {
    const template = this.deleteModalTemplate();
    if (template) {
      this.activeModalRef = this.modalService.open(template, { centered: true });
    }
  }

  public closeDeleteModal(): void {
    this.activeModalRef?.dismiss();
  }

  public async deleteEquipment(): Promise<void> {
    const eq = this.equipment();
    if (!eq) {
      return;
    }

    try {
      await firstValueFrom(this.api.deleteEquipment(eq.id));
      this.activeModalRef?.close();
      this.router.navigate(['/equipment']);
    } catch (err) {
      console.error('Failed to delete equipment:', err);
      this.error.set('Failed to delete equipment. Please try again.');
      this.activeModalRef?.dismiss();
    }
  }

  public goBack(): void {
    this.router.navigate(['/equipment']);
  }
}
