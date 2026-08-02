import {
  ChangeDetectionStrategy,
  Component,
  computed,
  effect,
  inject,
  OnInit,
  signal,
} from '@angular/core';
import { DomSanitizer, SafeHtml } from '@angular/platform-browser';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Api } from '../../../../core/services/api';
import { AppConfig } from '../../../../core/services/app-config';
import { PublicLayout } from '../../../../layouts/public-layout/public-layout';
import { catchError, of } from 'rxjs';

@Component({
  selector: 'app-legal-document',
  imports: [PublicLayout],
  templateUrl: './legal-document.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class LegalDocumentPage implements OnInit {
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private api = inject(Api);
  private appConfig = inject(AppConfig);
  private translate = inject(TranslateService);
  private sanitizer = inject(DomSanitizer);

  public readonly docType = signal<'legal-notice' | 'privacy'>('legal-notice');
  public readonly htmlContent = signal<string>('');

  public readonly sanitizedContent = computed<SafeHtml>(() => {
    return this.sanitizer.bypassSecurityTrustHtml(this.htmlContent());
  });

  public constructor() {
    effect(() => {
      const lang = this.translate.currentLang() || 'en';
      const type = this.docType();
      this.loadDocumentFor(type, lang);
    });
  }

  public ngOnInit(): void {
    this.route.data.subscribe((data) => {
      const type = data['type'] || 'legal-notice';
      this.docType.set(type);
    });
  }

  private loadDocumentFor(type: 'legal-notice' | 'privacy', lang: string): void {
    const availableLangs =
      type === 'legal-notice'
        ? this.appConfig.getLegalNoticeLanguages()
        : this.appConfig.getPrivacyLanguages();

    if (availableLangs.length === 0) {
      this.router.navigate(['/']);
      return;
    }

    const request$ =
      type === 'legal-notice' ? this.api.getLegalNotice(lang) : this.api.getPrivacyPolicy(lang);

    request$
      .pipe(
        catchError((err) => {
          console.error(`Failed to load ${type} for language ${lang}`, err);
          return of('<h3>Error loading document</h3>');
        }),
      )
      .subscribe((html) => {
        this.htmlContent.set(html);
      });
  }
}
