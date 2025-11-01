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

  readonly workout = signal<WorkoutDetail | null>(null);
  readonly loading = signal(true);
  readonly error = signal<string | null>(null);

  ngOnInit() {
    const uuid = this.route.snapshot.paramMap.get('uuid');
    if (uuid) {
      this.loadWorkout(uuid);
    } else {
      this.error.set('Invalid workout link');
      this.loading.set(false);
    }
  }

  loadWorkout(uuid: string) {
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

  readonly hasMapData = computed(() => {
    const workout = this.workout();
    return !!(workout?.map_data?.details?.position && workout.map_data.details.position.length > 0);
  });

  readonly selectedPosition = computed(() => 0);

  formatDuration(seconds: number): string {
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
