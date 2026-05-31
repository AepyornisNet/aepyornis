import { Injectable, Pipe, PipeTransform } from '@angular/core';

@Injectable({
  providedIn: 'root',
})
@Pipe({
  name: 'formatDate',
})
export class FormatDatePipe implements PipeTransform {
  public transform(value: string): string {
    return new Date(value).toLocaleDateString();
  }
}
