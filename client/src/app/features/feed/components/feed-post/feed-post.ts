import { ChangeDetectionStrategy, Component, effect, inject, input, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { TranslatePipe } from '@ngx-translate/core';
import { firstValueFrom } from 'rxjs';

import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { Avatar } from '../../../../core/components/avatar/avatar';
import { Api } from '../../../../core/services/api';
import { Workout, WorkoutReply } from '../../../../core/types/workout';

@Component({
  selector: 'app-feed-post',
  imports: [FormsModule, RouterLink, AppIcon, Avatar, TranslatePipe],
  templateUrl: './feed-post.html',
  styleUrl: './feed-post.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class FeedPost {
  private api = inject(Api);

  public readonly workout = input.required<Workout>();

  public readonly workoutState = signal<Workout | null>(null);
  public readonly commentsExpanded = signal(false);
  public readonly loadingReplies = signal(false);
  public readonly replyDraft = signal('');
  public readonly isReplying = signal(false);
  public readonly isLiking = signal(false);
  public readonly replies = signal<WorkoutReply[]>([]);

  public constructor() {
    effect(() => {
      this.workoutState.set(this.workout());
      this.commentsExpanded.set(false);
      this.replies.set([]);
      this.replyDraft.set('');
    });
  }

  public formatDate(dateString: string): string {
    return new Date(dateString).toLocaleDateString();
  }

  public formatDistance(distance: number): string {
    return (distance / 1000).toFixed(2);
  }

  public formatDuration(seconds: number): string {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    if (hours > 0) {
      return `${hours}h ${minutes}m`;
    }
    return `${minutes}m`;
  }

  public formatWeight(weight: number): string {
    return weight.toFixed(1);
  }

  public getWorkoutAuthorName(workout: Workout): string {
    const name = workout.user?.name?.trim();
    if (name) {
      return name;
    }

    const handle = this.formatUserHandle(workout.user);
    if (handle) {
      return handle;
    }

    return 'Unknown';
  }

  public getAuthorName(reply: WorkoutReply): string {
    const userName = reply.user?.name?.trim();
    if (userName) {
      return userName;
    }

    const userHandle = this.formatUserHandle(reply.user);
    if (userHandle) {
      return userHandle;
    }
    if (reply.actor_name) {
      return reply.actor_name;
    }

    const parsed = this.parseActorIri(reply.actor_iri);
    if (parsed?.username) {
      return `${parsed.username}@${parsed.host}`;
    }

    if (reply.actor_iri) {
      return reply.actor_iri;
    }
    return 'Unknown';
  }

  public getPublishDate(reply: WorkoutReply): string {
    const date = reply.published_at || reply.created_at;
    if (!date) {
      return '';
    }
    return new Date(date).toLocaleDateString();
  }

  public canSubmitReply(): boolean {
    return this.replyDraft().trim().length > 0 && !this.isReplying();
  }

  public async toggleComments(): Promise<void> {
    const isExpanded = this.commentsExpanded();
    this.commentsExpanded.set(!isExpanded);

    if (!isExpanded && this.replies().length === 0) {
      await this.loadReplies();
    }
  }

  public async likeWorkout(): Promise<void> {
    const workout = this.workoutState();
    if (!workout || this.isLiking()) {
      return;
    }

    this.isLiking.set(true);
    try {
      const response = await firstValueFrom(this.api.likeWorkout(workout.id));
      if (!response?.results) {
        return;
      }

      this.workoutState.update((current) =>
        current
          ? {
              ...current,
              liked_by_me: response.results.liked,
              likes_count: response.results.likes_count,
            }
          : current,
      );
    } catch (error) {
      console.error('Failed to like workout:', error);
    } finally {
      this.isLiking.set(false);
    }
  }

  public async submitReply(): Promise<void> {
    const workout = this.workoutState();
    const content = this.replyDraft().trim();
    if (!workout || !content || this.isReplying()) {
      return;
    }

    this.isReplying.set(true);
    try {
      const response = await firstValueFrom(this.api.createReply(workout.id, content));
      if (!response?.results) {
        return;
      }

      this.replies.update((current) => [response.results, ...current]);
      this.replyDraft.set('');
      this.workoutState.update((current) =>
        current
          ? {
              ...current,
              replies_count: (current.replies_count || 0) + 1,
            }
          : current,
      );
    } catch (error) {
      console.error('Failed to create reply:', error);
    } finally {
      this.isReplying.set(false);
    }
  }

  private async loadReplies(): Promise<void> {
    const workout = this.workoutState();
    if (!workout) {
      return;
    }

    this.loadingReplies.set(true);

    try {
      const response = await firstValueFrom(this.api.getWorkoutReplies(workout.id));

      const incomingReplies = response?.results || [];
      this.replies.set(incomingReplies);
    } catch (error) {
      console.error('Failed to load workout replies:', error);
    } finally {
      this.loadingReplies.set(false);
    }
  }

  private formatUserHandle(user?: WorkoutReply['user'] | Workout['user']): string {
    const username = user?.username?.trim();
    if (!username) {
      return '';
    }

    const domain = user?.domain?.trim();
    if (domain) {
      return `${username}@${domain}`;
    }

    return username;
  }

  private parseActorIri(actorIri?: string): { host: string; username: string } | null {
    if (!actorIri) {
      return null;
    }

    try {
      const url = new URL(actorIri);
      const segments = url.pathname.split('/').filter((segment) => segment.length > 0);

      let username = '';
      const usersIndex = segments.findIndex((segment) => segment === 'users' || segment === 'u');
      if (usersIndex >= 0 && usersIndex + 1 < segments.length) {
        username = segments[usersIndex + 1];
      } else {
        const mentionSegment = segments.find((segment) => segment.startsWith('@'));
        if (mentionSegment) {
          username = mentionSegment.slice(1);
        } else if (segments.length > 0) {
          username = segments[segments.length - 1];
        }
      }

      username = decodeURIComponent(username).replace(/^@+/, '').trim();

      return {
        host: url.host,
        username,
      };
    } catch {
      return null;
    }
  }
}
