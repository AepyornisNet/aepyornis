import {
  ChangeDetectionStrategy,
  Component,
  computed,
  effect,
  inject,
  OnInit,
  signal,
} from '@angular/core';

import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { TranslatePipe } from '@ngx-translate/core';
import { WorkoutMapComponent } from '../../components/workout-map/workout-map';
import { WorkoutChartComponent } from '../../components/workout-chart/workout-chart';
import { WorkoutBreakdownComponent } from '../../components/workout-breakdown/workout-breakdown';
import { WorkoutActions } from '../../components/workout-actions/workout-actions';
import { RouteSegmentMatchesComponent } from '../../components/route-segment-matches/route-segment-matches';
import { WorkoutClimbsComponent } from '../../components/workout-climbs/workout-climbs';
import { WorkoutZoneDistributionComponent } from '../../components/workout-zone-distribution/workout-zone-distribution';
import { WorkoutDetailDataService } from '../../services/workout-detail-data.service';
import { WorkoutDetailCoordinatorService } from '../../services/workout-detail-coordinator.service';
import { Workout } from '../../../../core/types/workout';
import { WorkoutRecordsComponent } from '../../components/workout-records/workout-records';
import { NgbNav, NgbNavContent, NgbNavItem, NgbNavLinkButton, NgbNavOutlet } from '@ng-bootstrap/ng-bootstrap';
import { hasWorkoutStatistics, WorkoutStatisticsComponent } from '../../components/workout-statistics/workout-statistics';
import { getSportLabel, getSportSubtypeLabel } from '../../../../core/i18n/sport-labels';

@Component({
  selector: 'app-workout-detail',
  imports: [
    AppIcon,
    WorkoutMapComponent,
    WorkoutChartComponent,
    WorkoutBreakdownComponent,
    WorkoutActions,
    RouteSegmentMatchesComponent,
    RouterLink,
    WorkoutClimbsComponent,
    WorkoutZoneDistributionComponent,
    WorkoutRecordsComponent,
    WorkoutStatisticsComponent,
    NgbNav,
    NgbNavOutlet,
    NgbNavItem,
    NgbNavLinkButton,
    NgbNavContent,
    TranslatePipe
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
  public readonly hasWorkoutStatisticsTab = computed(() =>
    hasWorkoutStatistics(this.dataService.workout()),
  );
  public readonly viewMode = signal(false);
  public readonly sportLabel = getSportLabel;
  public readonly sportSubtypeLabel = getSportSubtypeLabel;

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
    this.dataService.clearWorkout();

    const uuid = this.route.snapshot.paramMap.get('uuid');
    if (uuid) {
      this.viewMode.set(true);
      void this.dataService.loadPublicWorkout(uuid);
      return;
    }

    this.viewMode.set(false);

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
