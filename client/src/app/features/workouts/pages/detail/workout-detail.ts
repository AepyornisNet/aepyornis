import { Component, computed, effect, inject, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { TranslatePipe } from '@ngx-translate/core';
import { WorkoutMapComponent } from '../../components/workout-map/workout-map';
import { WorkoutChartComponent } from '../../components/workout-chart/workout-chart';
import { WorkoutBreakdownComponent } from '../../components/workout-breakdown/workout-breakdown';
import { WorkoutActionsComponent } from '../../components/workout-actions/workout-actions';
import { WorkoutDetailDataService } from '../../services/workout-detail-data.service';
import { WorkoutDetailCoordinatorService } from '../../services/workout-detail-coordinator.service';
import { Workout } from '../../../../core/types/workout';
import { User } from '../../../../core/services/user';

@Component({
  selector: 'app-workout-detail',
  imports: [
    CommonModule,
    RouterLink,
    AppIcon,
    WorkoutMapComponent,
    WorkoutChartComponent,
    WorkoutBreakdownComponent,
    WorkoutActionsComponent,
    TranslatePipe,
  ],
  templateUrl: './workout-detail.html',
  styleUrl: './workout-detail.scss',
})
export class WorkoutDetailPage implements OnInit {
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private userService = inject(User);

  // Inject services
  dataService = inject(WorkoutDetailDataService);
  coordinatorService = inject(WorkoutDetailCoordinatorService);

  // Check if socials are disabled from user profile
  readonly socialsDisabled = computed(() => {
    const userInfo = this.userService.getUserInfo()();
    return userInfo?.profile?.socials_disabled ?? false;
  });

  constructor() {
    // React to interval selection changes
    // The effect ensures that changes to the coordinator service's selectedInterval
    // are propagated to all child components
    effect(() => {
      // Trigger change detection when interval selection changes
      this.coordinatorService.selectedInterval();
    });
  }

  ngOnInit() {
    const id = this.route.snapshot.paramMap.get('id');
    if (id) {
      this.dataService.loadWorkout(parseInt(id, 10));
    } else {
      this.dataService.error.set('Invalid workout ID');
      this.dataService.loading.set(false);
    }
  }

  goBack() {
    this.router.navigate(['/workouts']);
  }

  onWorkoutUpdated(workout: Workout) {
    // Reload the workout to get the updated state
    this.dataService.loadWorkout(workout.id);
  }

  onWorkoutDeleted() {
    // Navigation is handled by the actions component
  }
}
