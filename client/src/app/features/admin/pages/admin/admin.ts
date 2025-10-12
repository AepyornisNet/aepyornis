import { Component, OnInit, signal, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { Api } from '../../../../core/services/api';
import { UserProfile, AppConfig } from '../../../../core/types/user';
import { FormBuilder, FormGroup, ReactiveFormsModule } from '@angular/forms';

@Component({
  selector: 'app-admin',
  imports: [CommonModule, RouterLink, AppIcon, ReactiveFormsModule],
  templateUrl: './admin.html',
  styleUrl: './admin.scss'
})
export class Admin implements OnInit {
  private api = inject(Api);
  private fb = inject(FormBuilder);

  users = signal<UserProfile[]>([]);
  loading = signal(true);
  error = signal<string | null>(null);
  savingConfig = signal(false);
  deleteConfirm = signal<number | null>(null);

  // Reactive form for app config
  configForm!: FormGroup;

  ngOnInit() {
    // Initialize config form
    this.configForm = this.fb.group({
      registration_disabled: [false],
      socials_disabled: [false]
    });

    this.loadData();
  }

  async loadData() {
    this.loading.set(true);
    this.error.set(null);

    try {
      // Load users
      const usersResponse = await firstValueFrom(this.api.getUsers());
      if (usersResponse?.results) {
        this.users.set(usersResponse.results);
      }

      // Load app info for config
      const appInfoResponse = await firstValueFrom(this.api.getAppInfo());
      if (appInfoResponse?.results) {
        this.configForm.patchValue({
          registration_disabled: appInfoResponse.results.registration_disabled,
          socials_disabled: appInfoResponse.results.socials_disabled
        });
      }
    } catch (err) {
      this.error.set('Failed to load data: ' + (err instanceof Error ? err.message : String(err)));
    } finally {
      this.loading.set(false);
    }
  }

  async saveConfig() {
    if (this.configForm.invalid) {
      return;
    }

    this.savingConfig.set(true);
    this.error.set(null);

    try {
      const config: AppConfig = this.configForm.value;
      const response = await firstValueFrom(this.api.updateAppConfig(config));
      if (response?.results) {
        this.configForm.patchValue({
          registration_disabled: response.results.registration_disabled,
          socials_disabled: response.results.socials_disabled
        });
      }
    } catch (err) {
      this.error.set('Failed to save config: ' + (err instanceof Error ? err.message : String(err)));
    } finally {
      this.savingConfig.set(false);
    }
  }

  confirmDelete(userId: number) {
    this.deleteConfirm.set(userId);
  }

  cancelDelete() {
    this.deleteConfirm.set(null);
  }

  async deleteUser(userId: number) {
    this.error.set(null);

    try {
      await firstValueFrom(this.api.deleteUser(userId));
      // Reload users after deletion
      await this.loadData();
      this.deleteConfirm.set(null);
    } catch (err) {
      this.error.set('Failed to delete user: ' + (err instanceof Error ? err.message : String(err)));
      this.deleteConfirm.set(null);
    }
  }
}
