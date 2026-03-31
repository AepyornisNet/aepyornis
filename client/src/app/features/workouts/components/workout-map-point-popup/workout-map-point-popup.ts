import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';

export type WorkoutMapPointPopupData = {
  time?: string;
  distance?: number;
  duration?: number;
  speed?: number | null;
  elevation?: number;
  slope?: number | null;
  heartRate?: number | null;
  power?: number | null;
  distanceUnit: string;
  speedUnit: string;
  elevationUnit: string;
};

@Component({
  selector: 'app-workout-map-point-popup',
  templateUrl: './workout-map-point-popup.html',
  styles: [
    `
      :host {
        display: block;
      }
    `,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class WorkoutMapPointPopupComponent {
  public readonly data = input.required<WorkoutMapPointPopupData>();

  public readonly formattedTime = computed(() => {
    const time = this.data().time;
    if (!time) {
      return null;
    }

    return new Date(time).toTimeString().substring(0, 5);
  });

  public readonly formattedDistance = computed(() => {
    const distance = this.data().distance;
    if (distance === undefined || distance === null) {
      return null;
    }
    return this.data().distanceUnit === 'mi' ? distance * 0.621371 : distance;
  });

  public readonly formattedDuration = computed(() => {
    const duration = this.data().duration;
    if (duration === undefined || duration === null) {
      return null;
    }

    const hours = Math.floor(duration / 3600);
    const minutes = Math.floor((duration % 3600) / 60);
    const seconds = Math.floor(duration % 60);

    if (hours > 0) {
      return `${hours}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
    }
    return `${minutes}:${seconds.toString().padStart(2, '0')}`;
  });

  public readonly formattedSpeed = computed(() => {
    const speed = this.data().speed;
    if (speed === undefined || speed === null) {
      return null;
    }
    return this.data().speedUnit === 'mph' ? speed * 2.23694 : speed * 3.6;
  });

  public readonly formattedElevation = computed(() => {
    const elevation = this.data().elevation;
    if (elevation === undefined || elevation === null) {
      return null;
    }
    return this.data().elevationUnit === 'ft' ? elevation * 3.28084 : elevation;
  });
}
