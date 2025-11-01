import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { RouterOutlet } from '@angular/router';
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
  private userService = inject(User);

  public readonly userName = computed(() => this.userService.getUserInfo()()?.name || '');
  public readonly sidebarOpen = signal(false);

  public handleLogout(): void {
    this.userService.logout();
  }

  public toggleSidebar(): void {
    this.sidebarOpen.update((open) => !open);
  }
}
