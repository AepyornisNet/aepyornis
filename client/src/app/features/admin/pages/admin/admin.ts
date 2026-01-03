import { ChangeDetectionStrategy, Component, inject, OnInit, signal } from '@angular/core';

import { RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { Api } from '../../../../core/services/api';
import { AppConfig, UserProfile } from '../../../../core/types/user';
import { FormBuilder, FormGroup, ReactiveFormsModule } from '@angular/forms';
import { TranslatePipe } from '@ngx-translate/core';

@Component({
  selector: 'app-admin',
  imports: [RouterLink, AppIcon, ReactiveFormsModule, TranslatePipe],
  templateUrl: './admin.html',
  styleUrl: './admin.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Admin implements OnInit {
  private api = inject(Api);
  private fb = inject(FormBuilder);

  public readonly users = signal<UserProfile[]>([]);
  public readonly loading = signal(true);
  public readonly error = signal<string | null>(null);
  public readonly savingConfig = signal(false);
  public readonly deleteConfirm = signal<number | null>(null);

  // Reactive form for app config
  public configForm!: FormGroup;

  public ngOnInit(): void {
    // Initialize config form
    this.configForm = this.fb.group({
      registration_disabled: [false],
      socials_disabled: [false],
    });

    this.loadData();
  }

  public async loadData(): Promise<void> {
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
          socials_disabled: appInfoResponse.results.socials_disabled,
        });
      }
    } catch (err) {
      this.error.set('Failed to load data: ' + (err instanceof Error ? err.message : String(err)));
    } finally {
      this.loading.set(false);
    }
  }

  public async saveConfig(): Promise<void> {
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
          socials_disabled: response.results.socials_disabled,
        });
      }
    } catch (err) {
      this.error.set(
        'Failed to save config: ' + (err instanceof Error ? err.message : String(err)),
      );
    } finally {
      this.savingConfig.set(false);
    }
  }

  public confirmDelete(userId: number): void {
    this.deleteConfirm.set(userId);
  }

  public cancelDelete(): void {
    this.deleteConfirm.set(null);
  }

  public async deleteUser(userId: number): Promise<void> {
    this.error.set(null);

    try {
      await firstValueFrom(this.api.deleteUser(userId));
      // Reload users after deletion
      await this.loadData();
      this.deleteConfirm.set(null);
    } catch (err) {
      this.error.set(
        'Failed to delete user: ' + (err instanceof Error ? err.message : String(err)),
      );
      this.deleteConfirm.set(null);
    }
  }
}
