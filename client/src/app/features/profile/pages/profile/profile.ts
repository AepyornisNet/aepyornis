import { ChangeDetectionStrategy, Component, inject, OnInit, signal } from '@angular/core';

import { firstValueFrom } from 'rxjs';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { FormBuilder, FormGroup, ReactiveFormsModule } from '@angular/forms';
import { Api } from '../../../../core/services/api';
import { AppConfig } from '../../../../core/services/app-config';
import { FollowRequest, FullUserProfile } from '../../../../core/types/user';
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
  protected appConfig = inject(AppConfig);
  private fb = inject(FormBuilder);
  private translate = inject(TranslateService);

  public readonly profile = signal<FullUserProfile | null>(null);
  public readonly loading = signal(true);
  public readonly saving = signal(false);
  public readonly error = signal<string | null>(null);
  public readonly successMessage = signal<string | null>(null);
  public readonly apiKeyVisible = signal(false);
  public readonly followRequests = signal<FollowRequest[]>([]);
  public readonly loadingFollowRequests = signal(false);
  public readonly acceptingRequestIds = signal<Record<number, boolean>>({});

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

        if (response.results.activity_pub) {
          await this.loadFollowRequests();
        } else {
          this.followRequests.set([]);
        }

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
        this.translate.instant('Failed to load profile: {{message}}', {
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
      const payload = {
        ...this.profileForm.value,
        auto_import_directory: this.appConfig.isAutoImportEnabled()
          ? this.profileForm.value.auto_import_directory
          : '',
      };

      const response = await firstValueFrom(this.api.updateProfile(payload));
      if (response?.results) {
        this.profile.set(response.results);
        this.successMessage.set(
          this.translate.instant('Profile updated successfully'),
        );
        // Clear success message after 3 seconds
        setTimeout(() => this.successMessage.set(null), 3000);
      }
    } catch (err) {
      this.error.set(
        this.translate.instant('Failed to save profile: {{message}}', {
          message: this.errorMessage(err),
        }),
      );
    } finally {
      this.saving.set(false);
    }
  }

  public async resetAPIKey(): Promise<void> {
    if (!confirm(this.translate.instant('Are you sure you want to generate a new API key? The old key will no longer work.'))) {
      return;
    }

    this.error.set(null);
    this.successMessage.set(null);

    try {
      const response = await firstValueFrom(this.api.resetAPIKey());
      if (response?.results) {
        this.successMessage.set(this.translate.instant('API key reset successfully'));
        // Reload profile to get new key
        await this.loadProfile();
      }
    } catch (err) {
      this.error.set(
        this.translate.instant('Failed to reset API key: {{message}}', {
          message: this.errorMessage(err),
        }),
      );
    }
  }

  public async refreshWorkouts(): Promise<void> {
    if (!confirm(this.translate.instant('Are you sure you want to refresh all your workouts? This may take several minutes.'))) {
      return;
    }

    this.error.set(null);
    this.successMessage.set(null);

    try {
      const response = await firstValueFrom(this.api.refreshWorkouts());
      if (response?.results) {
        this.successMessage.set(
          response.results.message ??
              this.translate.instant('Workouts refreshed'),
        );
      }
    } catch (err) {
      this.error.set(
        this.translate.instant('Failed to refresh workouts: {{message}}', {
          message: this.errorMessage(err),
        }),
      );
    }
  }

  public async enableActivityPub(): Promise<void> {
    if (!confirm(this.translate.instant('Enable ActivityPub for your account?'))) {
      return;
    }

    this.error.set(null);
    this.successMessage.set(null);

    try {
      const response = await firstValueFrom(this.api.enableActivityPub());
      if (response?.results) {
        this.successMessage.set(
          response.results.message ?? this.translate.instant('ActivityPub enabled'),
        );
        await this.loadProfile();
      }
    } catch (err) {
      this.error.set(
        this.translate.instant('Failed to enable ActivityPub: {{message}}', {
          message: this.errorMessage(err),
        }),
      );
    }
  }

  public async loadFollowRequests(): Promise<void> {
    this.loadingFollowRequests.set(true);

    try {
      const response = await firstValueFrom(this.api.getFollowRequests());
      this.followRequests.set(response?.results ?? []);
    } catch (err) {
      this.error.set(
        this.translate.instant('Failed to load follow requests: {{message}}', {
          message: this.errorMessage(err),
        }),
      );
    } finally {
      this.loadingFollowRequests.set(false);
    }
  }

  public async acceptFollowRequest(request: FollowRequest): Promise<void> {
    this.acceptingRequestIds.update((value) => ({ ...value, [request.id]: true }));

    try {
      await firstValueFrom(this.api.acceptFollowRequest(request.id));
      this.followRequests.update((value) => value.filter((item) => item.id !== request.id));
      this.successMessage.set(this.translate.instant('Follow request accepted'));
      setTimeout(() => this.successMessage.set(null), 3000);
    } catch (err) {
      this.error.set(
        this.translate.instant('Failed to accept follow request: {{message}}', {
          message: this.errorMessage(err),
        }),
      );
    } finally {
      this.acceptingRequestIds.update((value) => ({ ...value, [request.id]: false }));
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
          this.translate.instant('Copied to clipboard'),
        );
        setTimeout(() => this.successMessage.set(null), 2000);
      })
      .catch(() => {
        this.error.set(this.translate.instant('Failed to copy to clipboard'));
      });
  }

  private errorMessage(err: unknown): string {
    return err instanceof Error ? err.message : String(err);
  }
}
