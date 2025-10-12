import { Component, signal, LOCALE_ID, inject, input, output, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { AppIcon } from '../app-icon/app-icon';

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
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class Header {
  private localeId = inject(LOCALE_ID);

  // Input for user info and logout handler
  userName = input<string>();
  onLogout = input<() => void>();
  showSidebar = input<boolean>(false);
  
  // Output for sidebar toggle
  toggleSidebar = output<void>();

  selectedLanguage = signal('en');

  languages: Language[] = [
    { code: 'en', name: 'English', flag: '🇬🇧' },
    { code: 'de', name: 'Deutsch', flag: '🇩🇪' }
  ];

  constructor() {
    const localeId = this.localeId;

    // Set the current locale from Angular's LOCALE_ID
    this.selectedLanguage.set(localeId);
  }

  onLanguageChange(event: Event) {
    const select = event.target as HTMLSelectElement;
    const newLocale = select.value;

    if (newLocale !== this.localeId) {
      localStorage.setItem("locale", newLocale)
      window.location.reload()
    }
  }

  onToggleSidebar() {
    this.toggleSidebar.emit();
  }
}
