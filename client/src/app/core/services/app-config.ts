import { Injectable, signal, inject } from '@angular/core';
import { Api } from './api';
import { AppInfo } from '../../core/types/user';
import { catchError, of } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class AppConfig {
  private api = inject(Api);

  private appInfo = signal<AppInfo | null>(null);
  private loading = signal<boolean>(false);

  constructor() {
    this.loadAppInfo();
  }

  getAppInfo() {
    return this.appInfo.asReadonly();
  }

  isLoading(): boolean {
    return this.loading();
  }

  loadAppInfo() {
    this.loading.set(true);
    this.api.getAppInfo()
      .pipe(
        catchError(() => {
          console.error('Failed to load app info');
          return of(null);
        })
      )
      .subscribe(response => {
        this.loading.set(false);
        if (response && response.results) {
          this.appInfo.set(response.results);
        }
      });
  }

  isRegistrationDisabled(): boolean {
    return this.appInfo()?.registration_disabled ?? false;
  }

  isSocialsDisabled(): boolean {
    return this.appInfo()?.socials_disabled ?? false;
  }

  getVersion(): string {
    return this.appInfo()?.version ?? '';
  }

  getVersionSha(): string {
    return this.appInfo()?.version_sha ?? '';
  }
}
