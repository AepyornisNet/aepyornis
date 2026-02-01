import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { TranslateLoader } from '@ngx-translate/core';
import { mergeMap, Observable } from 'rxjs';
import PO from 'pofile';

@Injectable({ providedIn: 'root' })
export class PoTranslateLoader implements TranslateLoader {
  private http = inject(HttpClient);

  private prefix = './assets/i18n/';
  private suffix = '.po';

  public getTranslation(lang: string): Observable<Record<string, string>> {
    const path = `${this.prefix}${lang}${this.suffix}`;
    return this.http.get(path, { responseType: 'text' }).pipe(
      mergeMap(
        (source) =>
          new Observable<Record<string, string>>((observer) => {
            try {
              const po = PO.parse(source);
              const translations: Record<string, string> = {};

              for (const item of po.items) {
                if (!item.msgid) {
                  continue;
                }
                const rawValue = item.msgstr?.[0] ?? '';
                const value = rawValue.trim() ? rawValue : item.msgid;
                translations[item.msgid] = value;
              }

              observer.next(translations);
              observer.complete();
            } catch (error) {
              observer.error(error ?? new Error('Failed to parse PO file.'));
            }
          }),
      ),
    );
  }
}
