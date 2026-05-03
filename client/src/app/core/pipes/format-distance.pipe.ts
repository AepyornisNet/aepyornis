import { Pipe, PipeTransform } from '@angular/core';
@Pipe({
  name: 'formatDistance',
})
export class FormatDistancePipe implements PipeTransform {
  public transform(value: number): string {
    const distance = value || 0;
    return (distance / 1000).toFixed(2) + ` km`;
  }
}
