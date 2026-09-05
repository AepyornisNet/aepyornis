import { ChangeDetectionStrategy, Component, inject, OnInit, signal } from '@angular/core';
import { disabled, form, FormField, FormRoot } from '@angular/forms/signals';
import { TranslatePipe, TranslateService } from '@ngx-translate/core';
import { firstValueFrom } from 'rxjs';
import { Api } from '../../../../core/services/api';
import { AppConfig } from '../../../../core/types/user';

@Component({
  selector: 'app-admin-application-settings',
  imports: [FormField, FormRoot, TranslatePipe],
  templateUrl: './application-settings.html',
  styleUrl: './application-settings.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AdminApplicationSettings implements OnInit {
  private api = inject(Api);
  private translate = inject(TranslateService);

  public readonly loading = signal(true);
  public readonly savingConfig = signal(false);
  public readonly error = signal<string | null>(null);

  public readonly configModel = signal({
    registration_disabled: false,
    socials_disabled: false,
  });

  public readonly configForm = form(
    this.configModel,
    (s) => {
      disabled(s.registration_disabled, { when: () => this.savingConfig() });
      disabled(s.socials_disabled, { when: () => this.savingConfig() });
    },
    {
      submission: {
        action: () => this.saveConfig(),
      },
    },
  );

  public ngOnInit(): void {
    this.loadConfig();
  }

  public async loadConfig(): Promise<void> {
    this.loading.set(true);
    this.error.set(null);

    try {
      const appInfoResponse = await firstValueFrom(this.api.getAppInfo());
      if (appInfoResponse?.results) {
        this.configModel.set({
          registration_disabled: Boolean(appInfoResponse.results.registration_disabled),
          socials_disabled: Boolean(appInfoResponse.results.socials_disabled),
        });
      }
    } catch (err) {
      this.error.set(
        this.translate.instant('Failed to load settings: {{message}}', {
          message: err instanceof Error ? err.message : String(err),
        }),
      );
    } finally {
      this.loading.set(false);
    }
  }

  public async saveConfig(): Promise<void> {
    if (this.configForm().invalid()) {
      return;
    }

    this.savingConfig.set(true);
    this.error.set(null);

    try {
      const config: AppConfig = this.configModel() as AppConfig;
      const response = await firstValueFrom(this.api.updateAppConfig(config));
      if (response?.results) {
        this.configModel.set({
          registration_disabled: Boolean(response.results.registration_disabled),
          socials_disabled: Boolean(response.results.socials_disabled),
        });
      }
    } catch (err) {
      this.error.set(
        this.translate.instant('Failed to save config: {{message}}', {
          message: err instanceof Error ? err.message : String(err),
        }),
      );
    } finally {
      this.savingConfig.set(false);
    }
  }
}
