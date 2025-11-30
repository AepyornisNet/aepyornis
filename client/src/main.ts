import { provideZoneChangeDetection } from "@angular/core";
import { bootstrapApplication } from '@angular/platform-browser';
import { appConfig } from './app/app.config';

// Keep bootstrapping minimal. ngx-translate will be configured in app providers and initialized
// by services/components (header and user service) which react to stored/user-selected language.
import('./app/app').then((comp) =>
  bootstrapApplication(comp.App, {
    ...appConfig, providers: [provideZoneChangeDetection(), ...appConfig.providers],
  }),
);
