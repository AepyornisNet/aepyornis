import { inject, Pipe, PipeTransform } from '@angular/core';
import { User } from '../services/user';
import { metersToFeet } from '../config/units';
@Pipe({
  name: 'formatElevation',
})
export class FormatElevationPipe implements PipeTransform {
  private user = inject(User);

  public transform(meters: number | undefined | null): string {
    if (meters === undefined || meters === null) {
      return `-`;
    }
    const units = this.user.getUserInfo()()?.profile?.profile.preferred_units;

    if (!units || units.elevation === 'm') {
      return `${meters.toFixed(0)} m`;
    }

    return `${(meters * metersToFeet).toFixed(0)} ft`;
  }
}
