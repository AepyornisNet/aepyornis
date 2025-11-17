import {
  ChangeDetectionStrategy,
  Component,
  effect,
  inject,
  OnInit,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { TranslatePipe } from '@ngx-translate/core';
import { WorkoutMapComponent } from '../../components/workout-map/workout-map';
import { WorkoutChartComponent } from '../../components/workout-chart/workout-chart';
import { WorkoutBreakdownComponent } from '../../components/workout-breakdown/workout-breakdown';
import { WorkoutActions } from '../../components/workout-actions/workout-actions';
import { RouteSegmentMatchesComponent } from '../../components/route-segment-matches/route-segment-matches';
import { WorkoutClimbsComponent } from '../../components/workout-climbs/workout-climbs';
import { WorkoutDetailDataService } from '../../services/workout-detail-data.service';
import { WorkoutDetailCoordinatorService } from '../../services/workout-detail-coordinator.service';
import { Workout } from '../../../../core/types/workout';

@Component({
  selector: 'app-workout-detail',
  imports: [
    CommonModule,
    AppIcon,
    WorkoutMapComponent,
    WorkoutChartComponent,
    WorkoutBreakdownComponent,
    WorkoutActions,
    RouteSegmentMatchesComponent,
    WorkoutClimbsComponent,
    TranslatePipe,
  ],
  templateUrl: './workout-detail.html',
  styleUrl: './workout-detail.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class WorkoutDetailPage implements OnInit {
  private route = inject(ActivatedRoute);
  private router = inject(Router);

  // Inject services
  public dataService = inject(WorkoutDetailDataService);
  public coordinatorService = inject(WorkoutDetailCoordinatorService);

  public constructor() {
    // React to interval selection changes
    // The effect ensures that changes to the coordinator service's selectedInterval
    // are propagated to all child components
    effect(() => {
      // Trigger change detection when interval selection changes
      this.coordinatorService.selectedInterval();
    });
  }

  public ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('id');
    if (id) {
      this.dataService.loadWorkout(parseInt(id, 10));
    } else {
      this.dataService.error.set('Invalid workout ID');
      this.dataService.loading.set(false);
    }
  }

  public goBack(): void {
    this.router.navigate(['/workouts']);
  }

  public onWorkoutUpdated(workout: Workout): void {
    // Reload the workout to get the updated state
    this.dataService.loadWorkout(workout.id);
  }

  public onWorkoutDeleted(): void {
    // Navigation is handled by the actions component
  }
}
