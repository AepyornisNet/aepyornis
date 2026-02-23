import { Injectable } from '@angular/core';
import { MissingTranslationHandler, MissingTranslationHandlerParams } from '@ngx-translate/core';

@Injectable({
  providedIn: 'root',
})
export class InterpolatingMissingTranslationHandler implements MissingTranslationHandler {
  private readonly templateMatcher = /{{\s?([^{}\s]*)\s?}}/g;

  public handle(params: MissingTranslationHandlerParams): string {
    const key = params.key ?? '';
    const interpolateParams = params.interpolateParams as Record<string, unknown> | undefined;

    if (!interpolateParams) {
      return key;
    }

    return key.replace(this.templateMatcher, (substring: string, token: string) => {
      const value = interpolateParams[token];
      if (value === null || value === undefined) {
        return substring;
      }

      return String(value);
    });
  }
}
