import { ChangeDetectionStrategy, Component, inject, OnInit, signal } from '@angular/core';
import { FormBuilder, ReactiveFormsModule } from '@angular/forms';
import { TranslatePipe } from '@ngx-translate/core';
import { firstValueFrom } from 'rxjs';
import { Api } from '../../../../core/services/api';
import { AppConfig } from '../../../../core/types/user';

@Component({
  selector: 'app-admin-application-settings',
  imports: [ReactiveFormsModule, TranslatePipe],
  templateUrl: './application-settings.html',
  styleUrl: './application-settings.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AdminApplicationSettings implements OnInit {
  private api = inject(Api);
  private fb = inject(FormBuilder);

  public readonly loading = signal(true);
  public readonly savingConfig = signal(false);
  public readonly error = signal<string | null>(null);

  public readonly configForm = this.fb.group({
    registration_disabled: [false],
    socials_disabled: [false],
  });

  public ngOnInit(): void {
    this.loadConfig();
  }

  public async loadConfig(): Promise<void> {
    this.loading.set(true);
    this.error.set(null);

    try {
      const appInfoResponse = await firstValueFrom(this.api.getAppInfo());
      if (appInfoResponse?.results) {
        this.configForm.patchValue({
          registration_disabled: appInfoResponse.results.registration_disabled,
          socials_disabled: appInfoResponse.results.socials_disabled,
        });
      }
    } catch (err) {
      this.error.set(
        'Failed to load settings: ' + (err instanceof Error ? err.message : String(err)),
      );
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
      const config: AppConfig = this.configForm.value as AppConfig;
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
}
