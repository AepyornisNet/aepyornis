import { bootstrapApplication } from '@angular/platform-browser';
import { appConfig } from './app/app.config';
import { LOCALE_ID } from '@angular/core';
import { loadTranslations } from '@angular/localize';

async function initLanguage(locale: string): Promise<void> {
  if (locale === 'en') {
    return;
  }
  const json = await fetch('/locale/messages.' + locale + '.json')
    .then((r) => r.json())
    .then((r) => r?.translations || {})
    .catch(e => {
      console.error(`Error loading ${locale} translation:`, e);
      locale = 'en';
      return {};
    });

  // Initialize translation
  loadTranslations(json);
  $localize.locale = locale;

  // Load required locale module (needs to be adjusted for different locales)
  // const localeModule = await import(`../node_modules/@angular/common/locales/de`);
  // registerLocaleData(localeModule.default);
}

const appLang = localStorage.getItem('locale') || 'en';
initLanguage(appLang)
  .then(() => import('./app/app'))
  .then((comp) =>
    bootstrapApplication(comp.App, {
      ...appConfig,
      providers: [...appConfig.providers, { provide: LOCALE_ID, useValue: appLang }],
    }),
  );
