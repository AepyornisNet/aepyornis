import { inject, Injectable, Pipe, PipeTransform } from '@angular/core';
import { getWorkoutTypeConfig } from '../types/workout-types';
import { User } from '../services/user';
import {
  metersPerMinuteToMinutePerMile,
  metersPerSecondToKilometersPerHour,
  metersPerSecondToMilesPerHour,
} from '../config/units';

@Injectable({
  providedIn: 'root',
})
@Pipe({
  name: 'formatSpeed',
})
export class FormatSpeedPipe implements PipeTransform {
  private user = inject(User);
  private get speedUnit(): string {
    return this.user.getUserInfo()()?.profile?.profile.preferred_units.speed ?? `km/h`;
  }

  public convert(metersPerSecond: number, type?: string | null | undefined): number {
    if (metersPerSecond === 0) {
      return 0;
    }

    const workoutTypeConfig = getWorkoutTypeConfig(type ?? 'other');

    if (workoutTypeConfig?.pace) {
      const metersPerMinute = metersPerSecond * 60;
      if (this.speedUnit === 'km/h') {
        return 1000 / metersPerMinute;
      } else {
        return metersPerMinuteToMinutePerMile / metersPerMinute;
      }
    }
    if (this.speedUnit === 'km/h') {
      return metersPerSecond * metersPerSecondToKilometersPerHour;
    }

    return metersPerSecond * metersPerSecondToMilesPerHour;

  }

  public transform(
    metersPerSecond: number | null | undefined,
    type?: string | null | undefined,
  ): string {
    if (metersPerSecond === undefined || metersPerSecond === null) {
      return `-`;
    }

    const workoutTypeConfig = getWorkoutTypeConfig(type ?? 'other');

    if (workoutTypeConfig?.pace) {
      const pace_unit: string = this.speedUnit === 'km/h' ? `km` : `mi`;
      const pace = this.convert(metersPerSecond, type)
      const minutes = Math.floor(pace);
      const secs = Math.round((pace - minutes) * 60);
      return `${minutes}:${secs.toString().padStart(2, '0')} /${pace_unit}`;
    }

    return `${this.convert(metersPerSecond, type).toFixed(2)} ${this.speedUnit}`;
  }
}
