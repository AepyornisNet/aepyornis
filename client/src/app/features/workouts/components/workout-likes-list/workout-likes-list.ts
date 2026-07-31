import { ChangeDetectionStrategy, Component, input, output } from '@angular/core';
import { RouterLink } from '@angular/router';
import { TranslatePipe } from '@ngx-translate/core';

import { Avatar } from '../../../../core/components/avatar/avatar';
import { WorkoutLike } from '../../../../core/types/workout';

@Component({
  selector: 'app-workout-likes-list',
  imports: [RouterLink, Avatar, TranslatePipe],
  templateUrl: './workout-likes-list.html',
  styleUrl: './workout-likes-list.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class WorkoutLikesList {
  public readonly likes = input.required<WorkoutLike[]>();
  public readonly loading = input<boolean>(false);

  public readonly itemClick = output<void>();

  public getLikeAuthorName(like: WorkoutLike): string {
    const name = like.user?.name?.trim();
    if (name) {
      return name;
    }
    if (like.actor_name?.trim()) {
      return like.actor_name.trim();
    }
    const username = like.user?.username?.trim();
    if (username) {
      return username;
    }
    const parsed = this.parseActorIri(like.actor_iri);
    if (parsed?.username) {
      return `${parsed.username}@${parsed.host}`;
    }
    return 'User';
  }

  public getLikeAuthorHandle(like: WorkoutLike): string | null {
    if (like.user?.handle) {
      return like.user.handle.replace(/^@/, '');
    }
    if (like.user?.username?.trim()) {
      return like.user.username.trim();
    }
    const parsed = this.parseActorIri(like.actor_iri);
    if (parsed?.username) {
      return `${parsed.username}@${parsed.host}`;
    }
    return null;
  }

  public getLikeDisplayHandle(like: WorkoutLike): string | null {
    if (like.user?.handle) {
      return like.user.handle.startsWith('@') ? like.user.handle : `@${like.user.handle}`;
    }
    const handle = this.getLikeAuthorHandle(like);
    if (handle) {
      return handle.startsWith('@') ? handle : `@${handle}`;
    }
    return null;
  }

  public onItemClick(): void {
    this.itemClick.emit();
  }

  private parseActorIri(actorIri?: string): { host: string; username: string } | null {
    if (!actorIri) {
      return null;
    }
    try {
      const url = new URL(actorIri);
      const parts = url.pathname.split('/').filter(Boolean);
      const username = parts[parts.length - 1];
      if (username) {
        return { host: url.host, username };
      }
    } catch {
      // Ignore URL parse errors
    }
    return null;
  }
}
