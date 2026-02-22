import { ChangeDetectionStrategy, Component } from '@angular/core';

import { RecentActivity } from '../../components/recent-activity/recent-activity';

@Component({
  selector: 'app-feed',
  imports: [RecentActivity],
  templateUrl: './feed.html',
  styleUrl: './feed.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Feed {}
