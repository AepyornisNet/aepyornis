import { ChangeDetectionStrategy, Component, computed, inject, input, output } from '@angular/core';
import { RouterLink, RouterLinkActive } from '@angular/router';
import { CommonModule } from '@angular/common';
import { AppIcon } from '../app-icon/app-icon';
import { User } from '../../../core/services/user';
import { NgbTooltipModule } from '@ng-bootstrap/ng-bootstrap';

interface MenuItem {
  label: string;
  iconKey: string;
  route: string;
  adminOnly?: boolean;
}

@Component({
  selector: 'app-sidebar',
  imports: [RouterLink, RouterLinkActive, CommonModule, AppIcon, NgbTooltipModule],
  templateUrl: './sidebar.html',
  styleUrl: './sidebar.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Sidebar {
  private userService = inject(User);

  readonly isOpen = input<boolean>(false);
  sidebarToggle = output<void>();

  allMenuItems: MenuItem[] = [
    { label: $localize`Dashboard`, iconKey: 'dashboard', route: '/dashboard' },
    { label: $localize`Workouts`, iconKey: 'workout', route: '/workouts' },
    { label: $localize`Measurements`, iconKey: 'scale', route: '/measurements' },
    { label: $localize`Statistics`, iconKey: 'statistics', route: '/statistics' },
    { label: $localize`Heatmap`, iconKey: 'heatmap', route: '/heatmap' },
    { label: $localize`Route segments`, iconKey: 'route-segment', route: '/route-segments' },
    { label: $localize`Equipment`, iconKey: 'equipment', route: '/equipment' },
    { label: $localize`Profile`, iconKey: 'user-profile', route: '/profile' },
    { label: $localize`Admin`, iconKey: 'admin', route: '/admin', adminOnly: true },
  ];

  // Computed property to filter menu items based on user permissions
  readonly menuItems = computed(() => {
    const userInfo = this.userService.getUserInfo()();
    const isAdmin = userInfo?.profile?.admin ?? false;

    return this.allMenuItems.filter((item) => !item.adminOnly || isAdmin);
  });

  onToggle() {
    this.sidebarToggle.emit();
  }
}
