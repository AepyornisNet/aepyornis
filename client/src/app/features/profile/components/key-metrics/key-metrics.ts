import { ChangeDetectionStrategy, Component, input } from '@angular/core';

import { TranslatePipe } from '@ngx-translate/core';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { Totals } from '../../../../core/types/workout';
import { FormatDistance } from '../../../../core/pipes/format-distance.pipe';
import { FormatDuration } from '../../../../core/pipes/format-duration.pipe';
import { FormatElevation } from '../../../../core/pipes/format-elevation.pipe';
import {
  NgbNav,
  NgbNavContent,
  NgbNavItem,
  NgbNavLinkButton,
  NgbNavOutlet,
} from '@ng-bootstrap/ng-bootstrap';

@Component({
  selector: 'app-key-metrics',
  imports: [
    AppIcon,
    TranslatePipe,
    FormatDistance,
    FormatDuration,
    FormatElevation,
    NgbNav,
    NgbNavOutlet,
    NgbNavContent,
    NgbNavItem,
    NgbNavLinkButton,
  ],
  templateUrl: './key-metrics.html',
  styleUrl: './key-metrics.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class KeyMetrics {
  public readonly totals = input<Totals[] | []>([]);
}
