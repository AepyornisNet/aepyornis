import {
  ChangeDetectionStrategy,
  Component,
  computed,
  effect,
  inject,
  input,
  LOCALE_ID,
  output,
  signal,
} from '@angular/core';

import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { AppIcon } from '../app-icon/app-icon';
import { TranslatePipe, TranslateService } from '@ngx-translate/core';
import { AVAILABLE_LANGUAGES, Language } from '../../config/languages';
import { NgbDropdown, NgbDropdownMenu, NgbDropdownToggle } from '@ng-bootstrap/ng-bootstrap';
import { Api } from '../../services/api';
import { Notification } from '../../types/notification';

@Component({
  selector: 'app-header',
  imports: [
    FormsModule,
    RouterLink,
    AppIcon,
    NgbDropdown,
    NgbDropdownMenu,
    NgbDropdownToggle,
    TranslatePipe,
  ],
  templateUrl: './header.html',
  styleUrl: './header.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Header {
  private localeId = inject(LOCALE_ID);
  private translate = inject(TranslateService);
  private api = inject(Api);

  // Input for user info and logout handler
  public readonly userName = input<string>();
  public readonly isAdmin = input<boolean>(false);
  public readonly showSidebar = input<boolean>(false);

  // Output for sidebar toggle
  public readonly toggleSidebar = output<void>();
  public readonly logout = output<void>();

  public readonly selectedLanguage = signal('en');
  public readonly notifications = signal<Notification[]>([]);
  public readonly loadingNotifications = signal(false);
  public readonly unreadCount = computed(() => this.notifications().length);

  public languages: Language[] = AVAILABLE_LANGUAGES;

  public constructor() {
    const localeId = this.localeId;

    // Set the current locale from stored locale or Angular's LOCALE_ID
    const stored = localStorage.getItem('locale') || localeId;
    const initialLanguage = stored || 'en';
    this.selectedLanguage.set(initialLanguage);

    // Apply the initial language immediately
    this.translate.use(initialLanguage);

    effect(() => {
      localStorage.setItem('locale', this.selectedLanguage());
      this.translate.use(this.selectedLanguage());
    });

    effect(() => {
      if (this.userName()) {
        this.loadNotifications();
      }
    });
  }

  public onToggleSidebar(): void {
    this.toggleSidebar.emit();
  }

  public openNotifications(open: boolean): void {
    if (open) {
      this.loadNotifications();
    }
  }

  public loadNotifications(): void {
    this.loadingNotifications.set(true);
    this.api.getNotifications().subscribe({
      next: (resp) => {
        this.notifications.set(resp.results ?? []);
        this.loadingNotifications.set(false);
      },
      error: () => {
        this.loadingNotifications.set(false);
      },
    });
  }

  public markAsRead(item?: Notification): void {
    const ids = item ? [item.id] : undefined;
    this.api.markNotificationsAsRead(ids).subscribe({
      next: () => {
        if (item) {
          this.notifications.update((list) => list.filter((n) => n.id !== item.id));
        } else {
          this.notifications.set([]);
        }
      },
    });
  }

  public onNotificationClick(item: Notification): void {
    this.markAsRead(item);
  }

  public onMarkAsReadClick(item: Notification, event: Event): void {
    event.stopPropagation();
    event.preventDefault();
    this.markAsRead(item);
  }
}
