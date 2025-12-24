import { ChangeDetectionStrategy, Component } from '@angular/core';
import { RouterLink, RouterLinkActive } from '@angular/router';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';

@Component({
  selector: 'app-statistics-nav',
  standalone: true,
  imports: [RouterLink, RouterLinkActive, AppIcon],
  templateUrl: './statistics-nav.html',
  styleUrl: './statistics-nav.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class StatisticsNav {}
