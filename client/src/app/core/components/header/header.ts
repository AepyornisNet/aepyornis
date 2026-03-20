import {
  ChangeDetectionStrategy,
  Component,
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
import { NgbDropdownModule } from '@ng-bootstrap/ng-bootstrap';

@Component({
  selector: 'app-header',
  imports: [FormsModule, RouterLink, AppIcon, NgbDropdownModule, TranslatePipe],
  templateUrl: './header.html',
  styleUrl: './header.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Header {
  private localeId = inject(LOCALE_ID);
  private translate = inject(TranslateService);

  // Input for user info and logout handler
  public readonly userName = input<string>();
  public readonly isAdmin = input<boolean>(false);
  public readonly showSidebar = input<boolean>(false);

  // Output for sidebar toggle
  public readonly toggleSidebar = output<void>();
  public readonly logout = output<void>();

  public readonly selectedLanguage = signal('en');

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
  }

  public onToggleSidebar(): void {
    this.toggleSidebar.emit();
  }
}
