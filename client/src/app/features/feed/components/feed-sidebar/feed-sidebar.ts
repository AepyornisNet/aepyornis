import { ChangeDetectionStrategy, Component, input, output } from '@angular/core';

import { TranslatePipe } from '@ngx-translate/core';
import { Footer } from '../../../../core/components/footer/footer';
import { FeedCalendar } from '../feed-calendar/feed-calendar';

@Component({
  selector: 'app-feed-sidebar',
  imports: [TranslatePipe, Footer, FeedCalendar],
  templateUrl: './feed-sidebar.html',
  styleUrl: './feed-sidebar.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class FeedSidebar {
  public readonly feedScope = input<'following' | 'global'>('following');
  public readonly feedScopeChange = output<'following' | 'global'>();

  public setFeedScope(scope: 'following' | 'global'): void {
    if (scope === this.feedScope()) {
      return;
    }

    this.feedScopeChange.emit(scope);
  }
}
