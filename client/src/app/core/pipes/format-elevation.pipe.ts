import { Pipe, PipeTransform } from '@angular/core';
@Pipe({
  name: 'formatElevation',
})
export class FormatElevation implements PipeTransform {
  public transform(value: number): string {
    const elevation = value || 0;
    return elevation.toFixed(0) + ` m`;
  }
}
