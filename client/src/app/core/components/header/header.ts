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

type Language = {
  code: string;
  name: string;
  flag: string;
};

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
  public readonly userName = input<string>();
  public readonly showSidebar = input<boolean>(false);

  // Output for sidebar toggle
  public readonly toggleSidebar = output<void>();
  public readonly logout = output<void>();

  public readonly selectedLanguage = signal('en');

  public languages: Language[] = [
    { code: 'en', name: 'English', flag: '🇬🇧' },
    { code: 'de', name: 'Deutsch', flag: '🇩🇪' },
  ];

  public constructor() {
    const localeId = this.localeId;

    // Set the current locale from stored locale or Angular's LOCALE_ID
    const stored = localStorage.getItem('locale') || localeId;
    this.selectedLanguage.set(stored || 'en');
  }

  public onLanguageChange(event: Event): void {
    const select = event.target as HTMLSelectElement;
    const newLocale = select.value;
    if (newLocale !== this.selectedLanguage()) {
      localStorage.setItem('locale', newLocale);
      this.translate.use(newLocale);
      this.selectedLanguage.set(newLocale);
    }
  }

  public onToggleSidebar(): void {
    this.toggleSidebar.emit();
  }
}
