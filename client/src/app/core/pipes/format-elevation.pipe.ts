import { inject, Injectable, Pipe, PipeTransform } from '@angular/core';
import { User } from '../services/user';
import { metersToFeet } from '../config/units';

@Injectable({
  providedIn: 'root',
})
@Pipe({
  name: 'formatElevation',
})
export class FormatElevationPipe implements PipeTransform {
  private user = inject(User);
  private get elevationUnit(): string {
    return this.user.getUserInfo()()?.profile?.profile.preferred_units.elevation ?? `m`;
  }

  public convert(meters: number): number {
    if (this.elevationUnit === 'm') {
      return meters;
    }
    return meters * metersToFeet;
  }

  public transform(meters: number | undefined | null): string {
    if (meters === undefined || meters === null) {
      return `-`;
    }

    return `${this.convert(meters).toFixed(0)} ${this.elevationUnit}`;
  }
}
