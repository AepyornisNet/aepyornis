import { ChangeDetectionStrategy, Component, inject, OnInit, signal } from '@angular/core';

import { firstValueFrom } from 'rxjs';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { FormBuilder, FormGroup, ReactiveFormsModule } from '@angular/forms';
import { Api } from '../../../../core/services/api';
import { FullUserProfile } from '../../../../core/types/user';
import { TranslatePipe, TranslateService } from '@ngx-translate/core';

@Component({
  selector: 'app-profile',
  imports: [AppIcon, ReactiveFormsModule, TranslatePipe],
  templateUrl: './profile.html',
  styleUrl: './profile.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Profile implements OnInit {
  private api = inject(Api);
  private fb = inject(FormBuilder);
  private translate = inject(TranslateService);

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
      birthdate: [''],
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
          birthdate: response.results.birthdate ? response.results.birthdate.split('T')[0] : '',
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
        this.translate.instant('user.profile.load_profile_failed', {
          message: this.errorMessage(err),
        }),
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
        this.successMessage.set(
          this.translate.instant('user.profile.profile_updated_success'),
        );
        // Clear success message after 3 seconds
        setTimeout(() => this.successMessage.set(null), 3000);
      }
    } catch (err) {
      this.error.set(
        this.translate.instant('user.profile.save_profile_failed', {
          message: this.errorMessage(err),
        }),
      );
    } finally {
      this.saving.set(false);
    }
  }

  public async resetAPIKey(): Promise<void> {
    if (!confirm(this.translate.instant('user.profile.confirm_reset_api_key'))) {
      return;
    }

    this.error.set(null);
    this.successMessage.set(null);

    try {
      const response = await firstValueFrom(this.api.resetAPIKey());
      if (response?.results) {
        this.successMessage.set(this.translate.instant('user.profile.reset_api_key_success'));
        // Reload profile to get new key
        await this.loadProfile();
      }
    } catch (err) {
      this.error.set(
        this.translate.instant('user.profile.reset_api_key_failed', {
          message: this.errorMessage(err),
        }),
      );
    }
  }

  public async refreshWorkouts(): Promise<void> {
    if (!confirm(this.translate.instant('user.profile.confirm_refresh_workouts'))) {
      return;
    }

    this.error.set(null);
    this.successMessage.set(null);

    try {
      const response = await firstValueFrom(this.api.refreshWorkouts());
      if (response?.results) {
        this.successMessage.set(
          response.results.message ??
              this.translate.instant('user.profile.refresh_workouts_success'),
        );
      }
    } catch (err) {
      this.error.set(
        this.translate.instant('user.profile.refresh_workouts_failed', {
          message: this.errorMessage(err),
        }),
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
        this.successMessage.set(
          this.translate.instant('user.profile.copied_to_clipboard'),
        );
        setTimeout(() => this.successMessage.set(null), 2000);
      })
      .catch(() => {
        this.error.set(this.translate.instant('user.profile.copy_failed'));
      });
  }

  private errorMessage(err: unknown): string {
    return err instanceof Error ? err.message : String(err);
  }
}
