import { ChangeDetectionStrategy, Component } from '@angular/core';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { TranslatePipe } from '@ngx-translate/core';
import { NgbDropdownModule } from '@ng-bootstrap/ng-bootstrap';
import { WorkoutActions } from '../workout-actions/workout-actions';

@Component({
  selector: 'app-workout-list-actions',
  imports: [AppIcon, TranslatePipe, NgbDropdownModule],
  templateUrl: './workout-list-actions.html',
  styleUrl: './workout-list-actions.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class WorkoutListActions extends WorkoutActions {
  public view(): void {
    this.router.navigate(['/workouts', this.workout().id]);
  }
}
