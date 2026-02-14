import { inject, Injectable, signal } from '@angular/core';
import { Api } from './api';
import { UserProfile } from '../../core/types/user';
import { catchError, map, Observable, of, tap } from 'rxjs';
import { TranslateService } from '@ngx-translate/core';
import { Router } from '@angular/router';

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
  private api = inject(Api);
  private translate = inject(TranslateService);
  private router = inject(Router);

  private readonly userInfo = signal<UserInfo | null>(null);
  private readonly checkingAuth = signal<boolean>(false);

  public getUserInfo(): ReturnType<typeof this.userInfo.asReadonly> {
    return this.userInfo.asReadonly();
  }

  public isAuthenticated(): boolean {
    return this.userInfo()?.isAuthenticated ?? false;
  }

  public isCheckingAuth(): boolean {
    return this.checkingAuth();
  }

  public setAuthenticatedUser(profile: UserProfile): void {
    const user: UserInfo = {
      username: profile.username,
      name: profile.name || profile.username,
      isAuthenticated: true,
      profile,
    };

    this.userInfo.set(user);

    const lang = profile.language;
    if (lang) {
      this.translate.use(lang);
      localStorage.setItem('locale', lang);
    }
  }

  public checkAuthStatus(): Observable<UserProfile | null> {
    this.checkingAuth.set(true);
    return this.api
      .whoami()
      .pipe(
        catchError(() => {
          // User is not authenticated
          this.userInfo.set(null);
          return of(null);
        }),
        tap((response) => {
          this.checkingAuth.set(false);
          if (response && response.results) {
            this.setAuthenticatedUser(response.results);
          } else {
            this.userInfo.set(null);
          }
        }),
        map(respone => respone ? respone.results : null)
      );
  }

  public clearUser(): void {
    this.userInfo.set(null);
  }

  public logout(): void {
    this.api
      .signOut()
      .pipe(
        catchError(() => of(null)),
        tap(() => {
          this.clearUser();
          this.router.navigate(['/login']);
        }),
      )
      .subscribe();
  }
}
