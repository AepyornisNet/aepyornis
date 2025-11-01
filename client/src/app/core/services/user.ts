import { inject, Injectable, signal } from '@angular/core';
import { Router } from '@angular/router';
import { AUTH_LOGOUT_URL } from '../../core/types/auth';
import { Api } from './api';
import { UserProfile } from '../../core/types/user';
import { catchError, of } from 'rxjs';
import { TranslateService } from '@ngx-translate/core';

export type UserInfo = {
  username: string;
  name: string;
  isAuthenticated: boolean;
  profile?: UserProfile;
};

@Injectable({
  providedIn: 'root',
})
export class User {
  private router = inject(Router);
  private api = inject(Api);
  private translate = inject(TranslateService);

  private readonly userInfo = signal<UserInfo | null>(null);
  private readonly checkingAuth = signal<boolean>(false);

  public constructor() {
    this.checkAuthStatus();
  }

  public getUserInfo(): ReturnType<typeof this.userInfo.asReadonly> {
    return this.userInfo.asReadonly();
  }

  public isAuthenticated(): boolean {
    return this.userInfo()?.isAuthenticated ?? false;
  }

  public isCheckingAuth(): boolean {
    return this.checkingAuth();
  }

  public checkAuthStatus(): void {
    this.checkingAuth.set(true);
    this.api
      .whoami()
      .pipe(
        catchError(() => {
          // User is not authenticated
          this.userInfo.set(null);
          return of(null);
        }),
      )
      .subscribe((response) => {
        this.checkingAuth.set(false);
        if (response && response.results) {
          const user: UserInfo = {
            username: response.results.username,
            name: response.results.name || response.results.username,
            isAuthenticated: true,
            profile: response.results,
          };
          this.userInfo.set(user);
          // If user profile contains a language, use it for translations
          const lang = user.profile?.language;
          if (lang) {
            this.translate.use(lang);
            localStorage.setItem('locale', lang);
          }
        } else {
          this.userInfo.set(null);
        }
      });
  }

  public clearUser(): void {
    this.userInfo.set(null);
  }

  public logout(): void {
    // Clear local user info
    this.clearUser();

    window.location.href = AUTH_LOGOUT_URL;
  }
}
