import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  OnInit,
  signal,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { WorkoutMapComponent } from '../../components/workout-map/workout-map';
import { WorkoutChartComponent } from '../../components/workout-chart/workout-chart';
import { WorkoutBreakdownComponent } from '../../components/workout-breakdown/workout-breakdown';
import { TranslatePipe } from '@ngx-translate/core';
import { WorkoutDetailCoordinatorService } from '../../services/workout-detail-coordinator.service';
import { Api } from '../../../../core/services/api';
import { WorkoutDetail } from '../../../../core/types/workout';

@Component({
  selector: 'app-public-workout',
  imports: [
    CommonModule,
    RouterLink,
    AppIcon,
    WorkoutMapComponent,
    WorkoutChartComponent,
    WorkoutBreakdownComponent,
    TranslatePipe,
  ],
  providers: [WorkoutDetailCoordinatorService],
  templateUrl: './public-workout.html',
  styleUrl: './public-workout.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class PublicWorkout implements OnInit {
  private route = inject(ActivatedRoute);
  private api = inject(Api);
  public readonly workout = signal<WorkoutDetail | null>(null);
  public readonly loading = signal(true);
  public readonly error = signal<string | null>(null);

  public ngOnInit(): void {
    const uuid = this.route.snapshot.paramMap.get('uuid');
    if (uuid) {
      this.loadWorkout(uuid);
    } else {
      this.error.set('Invalid workout link');
      this.loading.set(false);
    }
  }

  public loadWorkout(uuid: string): void {
    this.loading.set(true);
    this.error.set(null);

    this.api.getPublicWorkout(uuid).subscribe({
      next: (response) => {
        if (response.errors && response.errors.length > 0) {
          this.error.set('Failed to load workout. The link may have expired or been removed.');
          this.loading.set(false);
          return;
        }

        this.workout.set(response.results);
        this.loading.set(false);
      },
      error: (err) => {
        console.error('Failed to load public workout:', err);
        this.error.set('Failed to load workout. The link may have expired or been removed.');
        this.loading.set(false);
      },
    });
  }

  public readonly hasTrack = computed<boolean>(() => {
    const workout = this.workout();
    return !!(workout?.has_tracks && workout.map_data?.details?.position && workout.map_data.details.position.length > 0);
  });

  public readonly hasChartData = computed<boolean>(() => {
    const details = this.workout()?.map_data?.details;
    if (!details) {
      return false;
    }

    const baseLengths = [details.time?.length || 0, details.distance?.length || 0, details.duration?.length || 0, details.speed?.length || 0, details.elevation?.length || 0];
    const extra = details.extra_metrics && Object.values(details.extra_metrics).some((arr) => Array.isArray(arr) && arr.length > 0);
    return baseLengths.some((len) => len > 0) || !!extra;
  });

  public readonly selectedPosition = computed<number>(() => 0);

  public formatDuration(seconds: number): string {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const secs = seconds % 60;

    if (hours > 0) {
      return `${hours}h ${minutes}m ${secs}s`;
    } else if (minutes > 0) {
      return `${minutes}m ${secs}s`;
    }
    return `${secs}s`;
  }
}
