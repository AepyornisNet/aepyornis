import { ChangeDetectionStrategy, Component, inject, OnInit, signal } from '@angular/core';

import { firstValueFrom } from 'rxjs';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { FormBuilder, FormGroup, ReactiveFormsModule } from '@angular/forms';
import { Api } from '../../../../core/services/api';
import { FullUserProfile } from '../../../../core/types/user';

@Component({
  selector: 'app-profile',
  imports: [AppIcon, ReactiveFormsModule],
  templateUrl: './profile.html',
  styleUrl: './profile.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Profile implements OnInit {
  private api = inject(Api);
  private fb = inject(FormBuilder);

  public readonly profile = signal<FullUserProfile | null>(null);
  public readonly loading = signal(true);
  public readonly saving = signal(false);
  public readonly error = signal<string | null>(null);
  public readonly successMessage = signal<string | null>(null);
  public readonly apiKeyVisible = signal(false);

  // Reactive form
  public profileForm!: FormGroup;

  public ngOnInit(): void {
    // Initialize form
    this.profileForm = this.fb.group({
      api_active: [false],
      totals_show: ['all'],
      timezone: ['UTC'],
      language: ['browser'],
      theme: ['browser'],
      auto_import_directory: [''],
      prefer_full_date: [false],
      socials_disabled: [false],
      preferred_units: this.fb.group({
        speed: ['km/h'],
        distance: ['km'],
        elevation: ['m'],
        weight: ['kg'],
        height: ['cm'],
      }),
    });

    this.loadProfile();
  }

  public async loadProfile(): Promise<void> {
    this.loading.set(true);
    this.error.set(null);

    try {
      const response = await firstValueFrom(this.api.getProfile());
      if (response?.results) {
        this.profile.set(response.results);
        // Update form with loaded profile data
        this.profileForm.patchValue({
          api_active: response.results.profile.api_active,
          totals_show: response.results.profile.totals_show,
          timezone: response.results.profile.timezone,
          language: response.results.profile.language,
          theme: response.results.profile.theme,
          auto_import_directory: response.results.profile.auto_import_directory,
          prefer_full_date: response.results.profile.prefer_full_date,
          socials_disabled: response.results.profile.socials_disabled,
          preferred_units: response.results.profile.preferred_units,
        });
      }
    } catch (err) {
      this.error.set(
        'Failed to load profile: ' + (err instanceof Error ? err.message : String(err)),
      );
    } finally {
      this.loading.set(false);
    }
  }

  public async saveProfile(): Promise<void> {
    if (this.profileForm.invalid) {
      return;
    }

    this.saving.set(true);
    this.error.set(null);
    this.successMessage.set(null);

    try {
      const response = await firstValueFrom(this.api.updateProfile(this.profileForm.value));
      if (response?.results) {
        this.profile.set(response.results);
        this.successMessage.set('Profile updated successfully');
        // Clear success message after 3 seconds
        setTimeout(() => this.successMessage.set(null), 3000);
      }
    } catch (err) {
      this.error.set(
        'Failed to save profile: ' + (err instanceof Error ? err.message : String(err)),
      );
    } finally {
      this.saving.set(false);
    }
  }

  public async resetAPIKey(): Promise<void> {
    if (
      !confirm('Are you sure you want to generate a new API key? The old key will no longer work.')
    ) {
      return;
    }

    this.error.set(null);
    this.successMessage.set(null);

    try {
      const response = await firstValueFrom(this.api.resetAPIKey());
      if (response?.results) {
        this.successMessage.set('API key reset successfully');
        // Reload profile to get new key
        await this.loadProfile();
      }
    } catch (err) {
      this.error.set(
        'Failed to reset API key: ' + (err instanceof Error ? err.message : String(err)),
      );
    }
  }

  public async refreshWorkouts(): Promise<void> {
    if (
      !confirm('Are you sure you want to refresh all your workouts? This may take several minutes.')
    ) {
      return;
    }

    this.error.set(null);
    this.successMessage.set(null);

    try {
      const response = await firstValueFrom(this.api.refreshWorkouts());
      if (response?.results) {
        this.successMessage.set(response.results.message);
      }
    } catch (err) {
      this.error.set(
        'Failed to refresh workouts: ' + (err instanceof Error ? err.message : String(err)),
      );
    }
  }

  public toggleAPIKeyVisibility(): void {
    this.apiKeyVisible.set(!this.apiKeyVisible());
  }

  public copyToClipboard(text: string): void {
    navigator.clipboard
      .writeText(text)
      .then(() => {
        this.successMessage.set('Copied to clipboard');
        setTimeout(() => this.successMessage.set(null), 2000);
      })
      .catch(() => {
        this.error.set('Failed to copy to clipboard');
      });
  }
}
