import { Pipe, PipeTransform } from '@angular/core';
@Pipe({
  name: 'formatElevation',
})
export class FormatElevationPipe implements PipeTransform {
  public transform(value: number | undefined | null): string {
    if (value === undefined || value === null) {
      return `-`;
    }
    const elevation = value || 0;
    return elevation.toFixed(0) + ` m`;
  }
}
