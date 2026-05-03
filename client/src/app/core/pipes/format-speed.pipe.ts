import { Pipe, PipeTransform } from '@angular/core';
import { getWorkoutTypeConfig } from '../types/workout-types';
@Pipe({
  name: 'formatSpeed',
})
export class FormatSpeedPipe implements PipeTransform {
  public transform(value: number | null | undefined, type?: string): string {
    if (value === undefined || value === null) {
      return `-`;
    }
    const workoutTypeConfig = getWorkoutTypeConfig(type ?? 'other');
    if (workoutTypeConfig?.pace) {
      const pace = 1000 / value / 60;
      const minutes = Math.floor(pace);
      const secs = Math.round((pace - minutes) * 60);
      return minutes + `:` + secs + ` /km`;
    }
    const speed = (value * 3.6).toFixed(2);
    return speed + ` km/h`;
  }
}
