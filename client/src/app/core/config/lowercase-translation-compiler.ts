import { Injectable } from '@angular/core';
import { TranslateCompiler, TranslationObject } from '@ngx-translate/core';

@Injectable()
export class LowercaseTranslationCompiler extends TranslateCompiler {
  compile(value: string): string {
    return value;
  }

  compileTranslations(translations: TranslationObject, lang: string): TranslationObject {
    const compiled: TranslationObject = {};

    for (const key in translations) {
      if (Object.prototype.hasOwnProperty.call(translations, key)) {
        const value = translations[key];
        if (typeof value === 'object' && value !== null) {
          compiled[key.toLowerCase()] = this.compileTranslations(value as TranslationObject, lang);
        } else if (typeof value === 'string') {
          compiled[key.toLowerCase()] = this.compile(value);
        }
      }
    }

    return compiled;
  }
}
