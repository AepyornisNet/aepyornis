import { ChangeDetectionStrategy, Component, inject, OnInit, signal } from '@angular/core';
import { form, FormField, FormRoot } from '@angular/forms/signals';
import { TranslatePipe, TranslateService } from '@ngx-translate/core';
import { SwPush } from '@angular/service-worker';
import { AppConfig } from '../../../../core/services/app-config';
import { ProfileStore } from '../../services/profile-store';
import { NotificationProvider, NotificationType } from '../../../../core/types/user';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { Api } from '../../../../core/services/api';
import { firstValueFrom } from 'rxjs';
import {
  UserNotificationSettings,
  UserNotificationSettingsUpdate,
} from '../../../../core/types/notification';

type NotificationProviderOption = {
  key: NotificationProvider;
  label: string;
};

type NotificationTypeOption = {
  key: NotificationType;
  label: string;
  description: string;
};

@Component({
  selector: 'app-profile-notifications',
  imports: [AppIcon, FormField, FormRoot, TranslatePipe],
  templateUrl: './notifications.html',
  styleUrl: './notifications.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ProfileNotificationsPage implements OnInit {
  public readonly appConfig = inject(AppConfig);
  private api = inject(Api);
  private swPush = inject(SwPush);
  private translate = inject(TranslateService);

  public store = inject(ProfileStore);
  public readonly loadingNotificationSettings = signal(true);
  public readonly requestingWebPush = signal(false);
  public readonly savingNotificationSettings = signal(false);

  public readonly notificationModel = signal({
    follow_request: {
      database: true,
      mail: true,
      webpush: true,
    },
    workout_like: {
      database: true,
      mail: true,
      webpush: true,
    },
    workout_reply: {
      database: true,
      mail: true,
      webpush: true,
    },
  });

  public readonly notificationForm = form(this.notificationModel, () => {}, {
    submission: {
      action: this.saveNotificationSettings.bind(this)
    }
  });

  private readonly notificationProviderOptions: NotificationProviderOption[] = [
    { key: 'database', label: 'In-app' },
    { key: 'mail', label: 'Email' },
    { key: 'webpush', label: 'Push' },
  ];

  public readonly notificationTypes: NotificationTypeOption[] = [
    {
      key: 'follow_request',
      label: 'Follow requests',
      description: 'When someone asks to follow your private profile.',
    },
    {
      key: 'workout_reply',
      label: 'Workout comments',
      description: 'When someone comments on one of your workouts.',
    },
    {
      key: 'workout_like',
      label: 'Workout likes',
      description: 'When someone likes one of your workouts.',
    },
  ];

  public ngOnInit(): void {
    void this.loadNotificationSettings();
  }

  public availableNotificationProviders(): NotificationProviderOption[] {
    const providers = this.appConfig.getNotificationProviders();
    return this.notificationProviderOptions.filter((provider) => providers.includes(provider.key));
  }

  public canRequestWebPush(): boolean {
    return Boolean(
      this.appConfig.getNotificationProviders().includes('webpush') &&
      this.appConfig.getAppInfo()()?.webpush_public_key &&
      this.swPush.isEnabled,
    );
  }

  public async saveNotificationSettings(): Promise<void> {
    const providers = this.availableNotificationProviders();
    if (providers.length === 0) {
      return;
    }

    this.savingNotificationSettings.set(true);
    this.store.error.set(null);
    this.store.successMessage.set(null);

    try {
      await Promise.all(
        providers.map(async (provider) => {
          const payload = await this.payload(provider.key);
          return firstValueFrom(this.api.updateNotificationSettings(provider.key, payload));
        }),
      );
      this.store.successMessage.set(this.translate.instant('Notification settings updated'));
      setTimeout(() => this.store.successMessage.set(null), 3000);
    } catch (err) {
      this.store.error.set(
        this.translate.instant('Failed to save notification settings: {{message}}', {
          message: err instanceof Error ? err.message : String(err),
        }),
      );
    } finally {
      this.savingNotificationSettings.set(false);
    }
  }

  public async requestWebpush(): Promise<void> {
    const serverPublicKey = this.appConfig.getAppInfo()()?.webpush_public_key;
    if (!serverPublicKey) {
      this.store.error.set(this.translate.instant('Web push is not configured on this server'));
      return;
    }

    this.requestingWebPush.set(true);
    this.store.error.set(null);
    this.store.successMessage.set(null);

    try {
      const subscription = await this.swPush.requestSubscription({ serverPublicKey });
      await firstValueFrom(
        this.api.updateNotificationSettings(
          'webpush',
          await this.payload('webpush', JSON.stringify(subscription)),
        ),
      );
      this.store.successMessage.set(this.translate.instant('Push notifications enabled'));
      setTimeout(() => this.store.successMessage.set(null), 3000);
    } catch (err) {
      this.store.error.set(
        this.translate.instant('Failed to enable push notifications: {{message}}', {
          message: err instanceof Error ? err.message : String(err),
        }),
      );
    } finally {
      this.requestingWebPush.set(false);
    }
  }

  private async loadNotificationSettings(): Promise<void> {
    this.loadingNotificationSettings.set(true);
    this.store.error.set(null);

    try {
      const response = await firstValueFrom(this.api.getNotificationSettings());
      for (const settings of response.results ?? []) {
        this.patchProviderSettings(settings);
      }
    } catch (err) {
      this.store.error.set(
        this.translate.instant('Failed to load notification settings: {{message}}', {
          message: err instanceof Error ? err.message : String(err),
        }),
      );
    } finally {
      this.loadingNotificationSettings.set(false);
    }
  }

  private patchProviderSettings(settings: UserNotificationSettings): void {
    const provider = settings.method as NotificationProvider;

    this.notificationForm
      .follow_request[provider]().value.set(settings.follow_request);
    this.notificationForm
      .workout_like[provider]().value.set(settings.workout_like);
    this.notificationForm
      .workout_reply[provider]().value.set(settings.workout_reply);
  }

  private async payload(
    provider: NotificationProvider,
    methodSettings: string | null = null,
  ): Promise<UserNotificationSettingsUpdate> {
    return {
      method_settings:
        methodSettings !== null ? methodSettings : await this.currentMethodSettings(provider),
      follow_request: Boolean(this.notificationForm.follow_request[provider]().value()),
      workout_like: Boolean(this.notificationForm.workout_like[provider]().value()),
      workout_reply: Boolean(this.notificationForm.workout_reply[provider]().value()),
    };
  }

  private async currentMethodSettings(provider: NotificationProvider): Promise<string> {
    if (provider !== 'webpush' || !this.swPush.isEnabled) {
      return '';
    }

    const subscription = await firstValueFrom(this.swPush.subscription);
    return subscription ? JSON.stringify(subscription) : '';
  }
}
