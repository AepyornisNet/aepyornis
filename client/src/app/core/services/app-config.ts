import { inject, Injectable, signal } from '@angular/core';
import { Api } from './api';
import { AppInfo } from '../../core/types/user';
import { catchError, of } from 'rxjs';

@Injectable({
  providedIn: 'root',
})
export class AppConfig {
  private api = inject(Api);

  private readonly appInfo = signal<AppInfo | null>(null);
  private readonly loading = signal<boolean>(false);

  public constructor() {
    this.loadAppInfo();
  }

  public getAppInfo(): ReturnType<typeof this.appInfo.asReadonly> {
    return this.appInfo.asReadonly();
  }

  public isLoading(): boolean {
    return this.loading();
  }

  public loadAppInfo(): void {
    this.loading.set(true);
    this.api
      .getAppInfo()
      .pipe(
        catchError(() => {
          console.error('Failed to load app info');
          return of(null);
        }),
      )
      .subscribe((response) => {
        this.loading.set(false);
        if (response && response.results) {
          this.appInfo.set(response.results);
        }
      });
  }

  public isRegistrationDisabled(): boolean {
    return this.appInfo()?.registration_disabled ?? false;
  }

  public isSocialsDisabled(): boolean {
    return this.appInfo()?.socials_disabled ?? false;
  }

  public getVersion(): string {
    return this.appInfo()?.version ?? '';
  }

  public getVersionSha(): string {
    return this.appInfo()?.version_sha ?? '';
  }
}
