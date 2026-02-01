import {
  ApplicationConfig,
  inject,
  provideAppInitializer,
  provideBrowserGlobalErrorListeners,
  provideZonelessChangeDetection,
} from '@angular/core';
import { provideRouter } from '@angular/router';
import { HttpClient, provideHttpClient, withFetch } from '@angular/common/http';
import { provideTranslateService, TranslateLoader } from '@ngx-translate/core';

import { routes } from './app.routes';
import { iconProviders } from './core/config/icon-providers';
import { PoTranslateLoader } from './core/i18n/po-translate-loader';
import { User } from './core/services/user';
import { AppConfig } from './core/services/app-config';

export const appConfig: ApplicationConfig = {
  providers: [
    provideBrowserGlobalErrorListeners(),
    provideZonelessChangeDetection(),
    provideRouter(routes),
    provideHttpClient(withFetch()),
    provideAppInitializer(() => {
      const appConfigService = inject(AppConfig);
      return appConfigService.loadAppInfo();
    }),
    provideAppInitializer(() => {
      const userService = inject(User);
      return userService.checkAuthStatus();
    }),
    provideTranslateService({
      loader: {
        provide: TranslateLoader,
        useClass: PoTranslateLoader,
        deps: [HttpClient],
      },
      fallbackLang: 'en',
      lang: 'en',
    }),
    iconProviders,
  ],
};
