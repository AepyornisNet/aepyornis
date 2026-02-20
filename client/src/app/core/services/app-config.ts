import { inject, Injectable, signal } from '@angular/core';
import { Api } from './api';
import { AppInfo } from '../../core/types/user';
import { catchError, map, Observable, of, tap } from 'rxjs';

@Injectable({
  providedIn: 'root',
})
export class AppConfig {
  private api = inject(Api);

  private readonly appInfo = signal<AppInfo | null>(null);
  private readonly loading = signal<boolean>(false);

  public getAppInfo(): ReturnType<typeof this.appInfo.asReadonly> {
    return this.appInfo.asReadonly();
  }

  public isLoading(): boolean {
    return this.loading();
  }

  public loadAppInfo(): Observable<AppInfo | null> {
    this.loading.set(true);
    return this.api
      .getAppInfo()
      .pipe(
        catchError(() => {
          console.error('Failed to load app info');
          return of(null);
        }),
        tap((response) => {
          this.loading.set(false);
          if (response && response.results) {
            this.appInfo.set(response.results);
          }
        }),
        map((response => response ? response.results : null))
      );
  }

  public isRegistrationDisabled(): boolean {
    return this.appInfo()?.registration_disabled ?? false;
  }

  public isSocialsDisabled(): boolean {
    return this.appInfo()?.socials_disabled ?? false;
  }

  public isAutoImportEnabled(): boolean {
    return this.appInfo()?.auto_import_enabled ?? false;
  }

  public getVersion(): string {
    return this.appInfo()?.version ?? '';
  }

  public getVersionSha(): string {
    return this.appInfo()?.version_sha ?? '';
  }
}
