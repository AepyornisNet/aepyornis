import { Pipe, PipeTransform } from '@angular/core';
@Pipe({
  name: 'formatDate',
})
export class FormatDatePipe implements PipeTransform {
  public transform(value: string): string {
    return new Date(value).toLocaleDateString();
  }
}
