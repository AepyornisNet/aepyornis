import { ChangeDetectionStrategy, Component, computed, inject, input, output } from '@angular/core';
import { RouterLink, RouterLinkActive } from '@angular/router';
import { _ } from '@ngx-translate/core';

import { AppIcon } from '../app-icon/app-icon';
import { User } from '../../../core/services/user';
import { NgbTooltipModule } from '@ng-bootstrap/ng-bootstrap';
import { TranslatePipe } from '@ngx-translate/core';

type MenuItem = {
  label: string;
  iconKey: string;
  route: string;
  adminOnly?: boolean;
};

@Component({
  selector: 'app-sidebar',
  imports: [RouterLink, RouterLinkActive, AppIcon, NgbTooltipModule, TranslatePipe],
  templateUrl: './sidebar.html',
  styleUrl: './sidebar.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Sidebar {
  private userService = inject(User);

  public readonly isOpen = input<boolean>(false);
  public readonly sidebarToggle = output<void>();

  public allMenuItems: MenuItem[] = [
    { label: _('Dashboard'), iconKey: 'dashboard', route: '/dashboard' },
    { label: _('Workouts'), iconKey: 'workout', route: '/workouts' },
    { label: _('Measurements'), iconKey: 'scale', route: '/measurements' },
    { label: _('Statistics'), iconKey: 'statistics', route: '/statistics' },
    { label: _('Heatmap'), iconKey: 'heatmap', route: '/heatmap' },
    { label: _('Route segments'), iconKey: 'route-segment', route: '/route-segments' },
    { label: _('Equipment'), iconKey: 'equipment', route: '/equipment' },
    { label: _('Profile'), iconKey: 'user-profile', route: '/profile' },
    { label: _('Admin'), iconKey: 'admin', route: '/admin', adminOnly: true },
  ];

  // Computed property to filter menu items based on user permissions
  public readonly menuItems = computed(() => {
    const userInfo = this.userService.getUserInfo()();
    const isAdmin = userInfo?.profile?.admin ?? false;

    return this.allMenuItems.filter((item) => !item.adminOnly || isAdmin);
  });

  public onToggle(): void {
    this.sidebarToggle.emit();
  }
}
