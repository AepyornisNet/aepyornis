import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { TranslatePipe } from '@ngx-translate/core';
import { RouteSegmentMatch } from '../../../../core/types/workout';

@Component({
  selector: 'app-route-segment-matches',
  imports: [CommonModule, TranslatePipe],
  templateUrl: './route-segment-matches.html',
  styleUrl: './route-segment-matches.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class RouteSegmentMatchesComponent {
  public readonly matches = input.required<RouteSegmentMatch[]>();

  public formatDistance(meters: number): string {
    return (meters / 1000).toFixed(2);
  }
}
