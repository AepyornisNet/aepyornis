import { Pipe, PipeTransform } from '@angular/core';
@Pipe({
  name: 'formatDuration',
})
export class FormatDurationPipe implements PipeTransform {
  public transform(value: number): string {
    const duration = value || 0;
    const hours = Math.floor(duration / 3600);
    const minutes = Math.floor((duration % 3600) / 60);
    return `${hours}h ${minutes}m`;
  }
}
