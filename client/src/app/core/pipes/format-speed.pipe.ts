import { inject, Pipe, PipeTransform } from '@angular/core';
import { getWorkoutTypeConfig } from '../types/workout-types';
import { User } from '../services/user';
import {
  metersPerMinuteToMinutePerMile,
  metersPerSecondToKilometersPerHour,
  metersPerSecondToMilesPerHour,
} from '../config/units';
@Pipe({
  name: 'formatSpeed',
})
export class FormatSpeedPipe implements PipeTransform {
  private user = inject(User);

  public transform(metersPerSecond: number | null | undefined, type?: string): string {
    if (metersPerSecond === undefined || metersPerSecond === null) {
      return `-`;
    }

    const units = this.user.getUserInfo()()?.profile?.profile.preferred_units;
    const workoutTypeConfig = getWorkoutTypeConfig(type ?? 'other');

    if (workoutTypeConfig?.pace) {
      let pace: number;
      let pace_unit: string;
      const metersPerMinute = metersPerSecond * 60;
      if (!units || units.speed === 'km/h') {
        pace = 1000 / metersPerMinute;
        pace_unit = `km`;
      } else {
        pace = metersPerMinuteToMinutePerMile / metersPerMinute;
        pace_unit = `mi`;
      }
      const minutes = Math.floor(pace);
      const secs = Math.round((pace - minutes) * 60);
      return `${minutes}:${secs.toString().padStart(2, '0')} /${pace_unit}`;
    }

    if (!units || units.speed === 'km/h') {
      return `${(metersPerSecond * metersPerSecondToKilometersPerHour).toFixed(2)} km/h`;
    }

    return `${(metersPerSecond * metersPerSecondToMilesPerHour).toFixed(2)} mph`;
  }
}
