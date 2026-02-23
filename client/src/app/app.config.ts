import {
  ApplicationConfig,
  inject,
  provideAppInitializer,
  provideBrowserGlobalErrorListeners,
  provideZonelessChangeDetection,
} from '@angular/core';
import { provideRouter } from '@angular/router';
import { HttpClient, provideHttpClient, withFetch, withInterceptors } from '@angular/common/http';
import {
  provideMissingTranslationHandler,
  provideTranslateService,
  TranslateLoader,
} from '@ngx-translate/core';

import { routes } from './app.routes';
import { iconProviders } from './core/config/icon-providers';
import { PoTranslateLoader } from './core/i18n/po-translate-loader';
import { User } from './core/services/user';
import { AppConfig } from './core/services/app-config';
import { authRecoveryInterceptor } from './core/interceptors/auth-recovery-interceptor';
import { InterpolatingMissingTranslationHandler } from './core/i18n/interpolating-missing-translation-handler';

export const appConfig: ApplicationConfig = {
  providers: [
    provideBrowserGlobalErrorListeners(),
    provideZonelessChangeDetection(),
    provideRouter(routes),
    provideHttpClient(withFetch(), withInterceptors([authRecoveryInterceptor])),
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
      missingTranslationHandler: provideMissingTranslationHandler(
        InterpolatingMissingTranslationHandler,
      ),
      fallbackLang: 'en',
      lang: 'en',
    }),
    iconProviders,
  ],
};
