import {
  ChangeDetectionStrategy,
  Component,
  inject,
  input,
  LOCALE_ID,
  output,
  signal,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { AppIcon } from '../app-icon/app-icon';
import { TranslateService } from '@ngx-translate/core';

interface Language {
  code: string;
  name: string;
  flag: string;
}

@Component({
  selector: 'app-header',
  imports: [CommonModule, FormsModule, AppIcon],
  templateUrl: './header.html',
  styleUrl: './header.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Header {
  private localeId = inject(LOCALE_ID);
  private translate = inject(TranslateService);

  // Input for user info and logout handler
  readonly userName = input<string>();
  readonly onLogout = input<() => void>();
  readonly showSidebar = input<boolean>(false);

  // Output for sidebar toggle
  toggleSidebar = output<void>();

  readonly selectedLanguage = signal('en');

  languages: Language[] = [
    { code: 'en', name: 'English', flag: '🇬🇧' },
    { code: 'de', name: 'Deutsch', flag: '🇩🇪' },
  ];

  constructor() {
    const localeId = this.localeId;

    // Set the current locale from stored locale or Angular's LOCALE_ID
    const stored = localStorage.getItem('locale') || localeId;
    this.selectedLanguage.set(stored || 'en');
  }

  onLanguageChange(event: Event) {
    const select = event.target as HTMLSelectElement;
    const newLocale = select.value;
    if (newLocale !== this.selectedLanguage()) {
      localStorage.setItem('locale', newLocale);
      this.translate.use(newLocale);
      this.selectedLanguage.set(newLocale);
    }
  }

  onToggleSidebar() {
    this.toggleSidebar.emit();
  }
}
