import { ChangeDetectionStrategy, Component, signal } from '@angular/core';

import { RecentActivity } from '../../components/recent-activity/recent-activity';
import { FeedSidebar } from '../../components/feed-sidebar/feed-sidebar';

@Component({
  selector: 'app-feed',
  imports: [RecentActivity, FeedSidebar],
  templateUrl: './feed.html',
  styleUrl: './feed.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Feed {
  public readonly feedScope = signal<'following' | 'global'>('following');

  public setFeedScope(scope: 'following' | 'global'): void {
    this.feedScope.set(scope);
  }
}
