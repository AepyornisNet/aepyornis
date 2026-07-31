import { Injectable, signal } from '@angular/core';

export type ThemeOption = 'light' | 'dark' | 'browser';

@Injectable({
  providedIn: 'root',
})
export class ThemeService {
  private readonly currentThemeSetting = signal<ThemeOption>('browser');
  private readonly activeTheme = signal<'light' | 'dark'>('light');

  public constructor() {
    const savedTheme = localStorage.getItem('theme') as ThemeOption | null;
    const initialTheme: ThemeOption =
      savedTheme === 'light' || savedTheme === 'dark' || savedTheme === 'browser'
        ? savedTheme
        : 'browser';

    this.currentThemeSetting.set(initialTheme);
    this.initSystemListener();
    this.applyTheme();
  }

  public setTheme(theme?: string | null): void {
    const validTheme: ThemeOption =
      theme === 'light' || theme === 'dark' || theme === 'browser' ? theme : 'browser';

    localStorage.setItem('theme', validTheme);
    this.currentThemeSetting.set(validTheme);
    this.applyTheme();
  }

  public getThemeSetting(): ThemeOption {
    return this.currentThemeSetting();
  }

  public getActiveTheme(): 'light' | 'dark' {
    return this.activeTheme();
  }

  public isDarkMode(): boolean {
    return this.activeTheme() === 'dark';
  }

  private applyTheme(): void {
    const setting = this.currentThemeSetting();
    let effectiveTheme: 'light' | 'dark';

    if (setting === 'light') {
      effectiveTheme = 'light';
    } else if (setting === 'dark') {
      effectiveTheme = 'dark';
    } else {
      // 'browser' or system preference
      const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
      effectiveTheme = prefersDark ? 'dark' : 'light';
    }

    this.activeTheme.set(effectiveTheme);
    document.documentElement.setAttribute('data-bs-theme', effectiveTheme);
  }

  private initSystemListener(): void {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    mediaQuery.addEventListener('change', () => {
      if (this.currentThemeSetting() === 'browser') {
        this.applyTheme();
      }
    });
  }
}
