import { Injectable, Pipe, PipeTransform } from '@angular/core';

@Injectable({
  providedIn: 'root',
})
@Pipe({
  name: 'formatDate',
})
export class FormatDatePipe implements PipeTransform {
  public transform(value: string | undefined | null): string {
    if (!value) {
      return '';
    }
    return new Date(value).toLocaleDateString();
  }
}
