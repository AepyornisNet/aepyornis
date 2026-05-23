import { inject, Pipe, PipeTransform } from '@angular/core';
import { User } from '../services/user';
import { metersToMiles } from '../config/units';
@Pipe({
  name: 'formatDistance',
})
export class FormatDistancePipe implements PipeTransform {
  private user = inject(User);

  public transform(meters: number | null | undefined): string {
    const units = this.user.getUserInfo()()?.profile?.profile.preferred_units;

    if (meters === undefined || meters === null) {
      return '—';
    }

    if (!units || units.distance === 'km') {
      return `${(meters / 1000).toFixed(2)} km`;
    }

    return `${(meters * metersToMiles).toFixed(2)} mi`;
  }
}
