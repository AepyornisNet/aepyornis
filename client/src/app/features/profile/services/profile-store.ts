import { effect, inject, Injectable, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';
import { form, maxLength, minLength, required, validate } from '@angular/forms/signals';
import { Api } from '../../../core/services/api';
import { AppConfig } from '../../../core/services/app-config';
import {
  FollowRequest,
  FullUserProfile,
  HammerheadConnectionStatus,
} from '../../../core/types/user';
import { TranslateService } from '@ngx-translate/core';
import { ThemeService } from '../../../core/services/theme';

@Injectable()
export class ProfileStore {
  private api = inject(Api);
  public readonly appConfig = inject(AppConfig);
  private translate = inject(TranslateService);
  private themeService = inject(ThemeService);

  public readonly profile = signal<FullUserProfile | null>(null);
  public readonly loading = signal(true);
  public readonly saving = signal(false);
  public readonly error = signal<string | null>(null);
  public readonly successMessage = signal<string | null>(null);
  public readonly changingPassword = signal(false);
  public readonly apiKeyVisible = signal(false);
  public readonly followRequests = signal<FollowRequest[]>([]);
  public readonly loadingFollowRequests = signal(false);
  public readonly acceptingRequestIds = signal<Record<number, boolean>>({});
  public readonly hammerheadConnection = signal<HammerheadConnectionStatus | null>(null);
  public readonly loadingHammerhead = signal(false);
  public readonly connectingHammerhead = signal(false);
  public readonly disconnectingHammerhead = signal(false);

  public readonly profileModel = signal({
    name: '',
    birthdate: '',
    api_active: false,
    totals_show: 'all',
    timezone: 'UTC',
    language: 'browser',
    theme: 'browser',
    map_style: 'default',
    auto_import_directory: '',
    prefer_full_date: false,
    default_workout_visibility: '' as '' | 'followers' | 'public',
    preferred_units: {
      speed: 'km/h',
      distance: 'km',
      elevation: 'm',
      weight: 'kg',
      height: 'cm',
    },
  });

  public readonly profileForm = form(
    this.profileModel,
    (s) => {
      required(s.name);
    },
    {
      submission: {
        action: () => this.saveProfile(),
      },
    },
  );

  public readonly changePasswordModel = signal({
    current_password: '',
    new_password: '',
    confirm_password: '',
  });

  public readonly changePasswordForm = form(
    this.changePasswordModel,
    (s) => {
      required(s.current_password);
      required(s.new_password);
      minLength(s.new_password, 4);
      maxLength(s.new_password, 128);
      required(s.confirm_password);
      validate(s.confirm_password, ({ valueOf, value }) => {
        if (!value() || !valueOf(s.new_password)) {
          return undefined;
        }
        return value() === valueOf(s.new_password)
          ? undefined
          : {
              kind: 'passwordMismatch',
              message: this.translate.instant('Passwords do not match'),
            };
      });
    },
    {
      submission: {
        action: () => this.changePassword(),
      },
    },
  );

  public constructor() {
    effect(() => {
      const theme = this.profileModel().theme;
      if (theme) {
        this.themeService.setTheme(theme);
      }
    });

    effect(() => {
      const mapStyle = this.profileModel().map_style;
      if (mapStyle) {
        this.themeService.setMapStyle(mapStyle);
      }
    });
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

        this.profileModel.set({
          name: response.results.name || '',
          birthdate: response.results.birthdate ? response.results.birthdate.split('T')[0] : '',
          api_active: Boolean(response.results.profile.api_active),
          totals_show: response.results.profile.totals_show || 'all',
          timezone: response.results.profile.timezone || 'UTC',
          language: response.results.profile.language || 'browser',
          theme: response.results.profile.theme || 'browser',
          map_style: response.results.profile.map_style || 'default',
          auto_import_directory: response.results.profile.auto_import_directory || '',
          prefer_full_date: Boolean(response.results.profile.prefer_full_date),
          default_workout_visibility: (response.results.profile.default_workout_visibility ||
            '') as '' | 'followers' | 'public',
          preferred_units: {
            speed: response.results.profile.preferred_units?.speed || 'km/h',
            distance: response.results.profile.preferred_units?.distance || 'km',
            elevation: response.results.profile.preferred_units?.elevation || 'm',
            weight: response.results.profile.preferred_units?.weight || 'kg',
            height: response.results.profile.preferred_units?.height || 'cm',
          },
        });

        this.themeService.setTheme(response.results.profile.theme);
        this.themeService.setMapStyle(response.results.profile.map_style);
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
    if (this.profileForm().invalid()) {
      return;
    }

    this.saving.set(true);
    this.error.set(null);
    this.successMessage.set(null);

    try {
      const payload = {
        ...this.profileModel(),
        auto_import_directory: this.appConfig.isAutoImportEnabled()
          ? this.profileModel().auto_import_directory
          : '',
      };

      const response = await firstValueFrom(this.api.updateProfile(payload));
      if (response?.results) {
        this.profile.set(response.results);

        this.themeService.setTheme(response.results.profile.theme);
        this.themeService.setMapStyle(response.results.profile.map_style);

        // Apply the language change if it's not "browser"
        const newLang = response.results.profile.language;
        if (newLang && newLang !== 'browser') {
          this.translate.use(newLang);
          localStorage.setItem('locale', newLang);
        } else if (newLang === 'browser') {
          // Use browser language
          const browserLang =
            localStorage.getItem('locale') || this.translate.getBrowserLang() || 'en';
          this.translate.use(browserLang);
        }

        this.successMessage.set(this.translate.instant('Profile updated successfully'));
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

  public async changePassword(): Promise<void> {
    if (this.changePasswordForm().invalid()) {
      return;
    }

    const currentPassword = this.changePasswordModel().current_password;
    const newPassword = this.changePasswordModel().new_password;
    const confirmPassword = this.changePasswordModel().confirm_password;

    if (newPassword !== confirmPassword) {
      this.error.set(this.translate.instant('Passwords do not match'));
      return;
    }

    this.changingPassword.set(true);
    this.error.set(null);
    this.successMessage.set(null);

    try {
      const response = await firstValueFrom(
        this.api.changePassword({
          current_password: currentPassword,
          new_password: newPassword,
        }),
      );

      this.successMessage.set(
        response?.results?.message ?? this.translate.instant('Password changed successfully'),
      );
      this.changePasswordModel.set({
        current_password: '',
        new_password: '',
        confirm_password: '',
      });
    } catch (err) {
      this.error.set(
        this.translate.instant('Failed to change password: {{message}}', {
          message: this.errorMessage(err),
        }),
      );
    } finally {
      this.changingPassword.set(false);
    }
  }

  public async resetAPIKey(): Promise<void> {
    if (
      !confirm(
        this.translate.instant(
          'Are you sure you want to generate a new API key? The old key will no longer work.',
        ),
      )
    ) {
      return;
    }

    this.error.set(null);
    this.successMessage.set(null);

    try {
      const response = await firstValueFrom(this.api.resetAPIKey());
      if (response?.results) {
        this.successMessage.set(this.translate.instant('API key reset successfully'));
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
    if (
      !confirm(
        this.translate.instant(
          'Are you sure you want to refresh all your workouts? This may take several minutes.',
        ),
      )
    ) {
      return;
    }

    this.error.set(null);
    this.successMessage.set(null);

    try {
      const response = await firstValueFrom(this.api.refreshWorkouts());
      if (response?.results) {
        this.successMessage.set(
          response.results.message ?? this.translate.instant('Workouts refreshed'),
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

  public async loadHammerheadConnection(): Promise<void> {
    this.loadingHammerhead.set(true);

    try {
      const response = await firstValueFrom(this.api.getHammerheadConnection());
      this.hammerheadConnection.set(response?.results ?? { connected: false });
    } catch (err) {
      this.error.set(
        this.translate.instant('Failed to load Hammerhead connection: {{message}}', {
          message: this.errorMessage(err),
        }),
      );
    } finally {
      this.loadingHammerhead.set(false);
    }
  }

  public async connectHammerhead(): Promise<void> {
    this.connectingHammerhead.set(true);
    this.error.set(null);

    try {
      const response = await firstValueFrom(this.api.connectHammerhead());
      const authorizeURL = response?.results?.authorize_url;
      if (!authorizeURL) {
        throw new Error(this.translate.instant('No authorize URL returned by server'));
      }

      window.location.href = authorizeURL;
    } catch (err) {
      this.error.set(
        this.translate.instant('Failed to start Hammerhead connection: {{message}}', {
          message: this.errorMessage(err),
        }),
      );
    } finally {
      this.connectingHammerhead.set(false);
    }
  }

  public async disconnectHammerhead(): Promise<void> {
    if (!confirm(this.translate.instant('Disconnect Hammerhead from your account?'))) {
      return;
    }

    this.disconnectingHammerhead.set(true);
    this.error.set(null);

    try {
      const response = await firstValueFrom(this.api.disconnectHammerhead());
      this.successMessage.set(
        response?.results?.message ?? this.translate.instant('Hammerhead disconnected'),
      );
      this.hammerheadConnection.set({ connected: false });
      setTimeout(() => this.successMessage.set(null), 3000);
    } catch (err) {
      this.error.set(
        this.translate.instant('Failed to disconnect Hammerhead: {{message}}', {
          message: this.errorMessage(err),
        }),
      );
    } finally {
      this.disconnectingHammerhead.set(false);
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
        this.successMessage.set(this.translate.instant('Copied to clipboard'));
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
