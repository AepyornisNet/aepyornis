import { inject, Injectable, Pipe, PipeTransform } from '@angular/core';
import { User } from '../services/user';
import { metersToMiles } from '../config/units';

@Injectable({
  providedIn: 'root',
})
@Pipe({
  name: 'formatDistance',
})
export class FormatDistancePipe implements PipeTransform {
  private user = inject(User);
  private get distanceUnit(): string {
    return this.user.getUserInfo()()?.profile?.profile.preferred_units.distance ?? `km`;
  }

  public convert(meters: number | null | undefined): number | null {
    if (meters === undefined || meters === null || Number.isNaN(meters)) {
      return null;
    }

    if (this.distanceUnit === 'km') {
      return meters / 1000;
    }

    return meters * metersToMiles;
  }

  public transform(meters: number | null | undefined): string {
    const value = this.convert(meters);
    if (value === null) {
      return '—';
    }

    if (this.distanceUnit === 'km') {
      return `${value.toFixed(2)} km`;
    }

    return `${value.toFixed(2)} mi`;
  }
}
