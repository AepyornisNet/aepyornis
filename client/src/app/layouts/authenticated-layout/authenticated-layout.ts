import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { NavigationEnd, Router, RouterOutlet } from '@angular/router';
import { filter } from 'rxjs';
import { Header } from '../../core/components/header/header';
import { Footer } from '../../core/components/footer/footer';
import { Sidebar } from '../../core/components/sidebar/sidebar';
import { User } from '../../core/services/user';

@Component({
  selector: 'app-authenticated-layout',
  imports: [RouterOutlet, Header, Footer, Sidebar],
  templateUrl: './authenticated-layout.html',
  styleUrl: './authenticated-layout.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AuthenticatedLayout {
  private static readonly SIDEBAR_STATE_KEY = 'sidebarOpen';

  private userService = inject(User);
  private router = inject(Router);

  public readonly userName = computed(() => this.userService.getUserInfo()()?.name || '');
  public readonly isAdmin = computed(
    () => this.userService.getUserInfo()()?.profile?.admin || false,
  );
  public readonly sidebarOpen = signal(AuthenticatedLayout.getInitialSidebarState());
  public readonly currentUrl = signal(this.router.url);
  public readonly isFeedRoute = computed(() => this.currentUrl().startsWith('/feed'));

  public constructor() {
    this.router.events
      .pipe(filter((event) => event instanceof NavigationEnd))
      .subscribe((event) => this.currentUrl.set(event.urlAfterRedirects));
  }

  private static getInitialSidebarState(): boolean {
    return localStorage.getItem(AuthenticatedLayout.SIDEBAR_STATE_KEY) === 'true';
  }

  public handleLogout(): void {
    this.userService.logout();
  }

  public toggleSidebar(): void {
    const nextState = !this.sidebarOpen();
    this.sidebarOpen.set(nextState);
    localStorage.setItem(AuthenticatedLayout.SIDEBAR_STATE_KEY, String(nextState));
  }
}
