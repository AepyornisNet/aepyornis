import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';

import { AppIcon } from '../app-icon/app-icon';

type AvatarUser = {
  name?: string | null;
  icon_url?: string | null;
};

@Component({
  selector: 'app-avatar',
  imports: [AppIcon],
  templateUrl: './avatar.html',
  styleUrl: './avatar.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Avatar {
  public readonly user = input<AvatarUser | null | undefined>(null);
  public readonly size = input<number>(44);
  public readonly iconSize = input<number>(20);

  public readonly hasImage = computed(() => {
    const source = this.user()?.icon_url;
    return !!(source && source.trim().length > 0);
  });

  public readonly initial = computed(() =>
    (this.user()?.name?.trim().charAt(0) || '?').toUpperCase(),
  );

  public readonly sizePx = computed(() => `${this.size()}px`);
}
