import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  input,
  linkedSignal,
  signal,
  TemplateRef,
  viewChild,
} from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { TranslatePipe } from '@ngx-translate/core';
import { firstValueFrom } from 'rxjs';
import { NgbModal } from '@ng-bootstrap/ng-bootstrap';

import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { Avatar } from '../../../../core/components/avatar/avatar';
import { Api } from '../../../../core/services/api';
import { User } from '../../../../core/services/user';
import { UserSummary } from '../../../../core/types/user';
import { Workout, WorkoutLike, WorkoutReply } from '../../../../core/types/workout';

import { WorkoutLikesList } from '../../../workouts/components/workout-likes-list/workout-likes-list';

@Component({
  selector: 'app-feed-post',
  imports: [FormsModule, RouterLink, AppIcon, Avatar, TranslatePipe, WorkoutLikesList],
  templateUrl: './feed-post.html',
  styleUrl: './feed-post.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class FeedPost {
  private api = inject(Api);
  private user = inject(User);
  private modalService = inject(NgbModal);

  public readonly workout = input.required<Workout>();

  public readonly likesModalTemplate = viewChild<TemplateRef<unknown>>('likesModal');

  public readonly workoutState = linkedSignal(() => this.workout());
  public readonly isLikedByMe = computed(() => this.workoutState()?.liked_by_me ?? false);
  public readonly likesCount = computed(() => this.workoutState()?.likes_count ?? 0);

  public readonly commentsExpanded = signal(false);
  public readonly loadingReplies = signal(false);
  public readonly replyDraft = signal('');
  public readonly isReplying = signal(false);
  public readonly isLiking = signal(false);
  public readonly replies = signal<WorkoutReply[]>([]);
  public readonly workoutLikes = linkedSignal<WorkoutLike[]>(
    () => this.workout().recent_likes || [],
  );
  public readonly loadingLikes = signal(false);

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

  public getWorkoutAuthorHandle(workout: Workout): string | null {
    return workout.user?.username?.trim() || null;
  }

  public isOwnWorkout(workout: Workout): boolean {
    const currentUser = this.user.getUserInfo()();
    if (!currentUser) {
      return false;
    }

    if (workout.user?.username && currentUser.username) {
      return workout.user.username === currentUser.username;
    }

    if (workout.user?.id && currentUser.profile?.id) {
      return workout.user.id === currentUser.profile.id;
    }

    return false;
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

  public getAuthorHandle(reply: WorkoutReply): string | null {
    if (reply.user?.username) {
      return reply.user.username.trim();
    }
    const parsed = this.parseActorIri(reply.actor_iri);
    return parsed?.username || null;
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

  public async loadLikes(): Promise<void> {
    const workout = this.workoutState();
    if (!workout || workout.likes_count === 0) {
      this.workoutLikes.set([]);
      return;
    }

    this.loadingLikes.set(true);
    try {
      const response = await firstValueFrom(this.api.getWorkoutLikes(workout.id));
      this.workoutLikes.set(response?.results || []);
    } catch (err) {
      console.error('Failed to load workout likes:', err);
    } finally {
      this.loadingLikes.set(false);
    }
  }

  public async openLikesModal(): Promise<void> {
    this.loadLikes();

    const template = this.likesModalTemplate();
    if (template) {
      this.modalService.open(template, { centered: true, scrollable: true });
    }
  }

  public getLikeAuthorName(like: WorkoutLike): string {
    const name = like.user?.name?.trim();
    if (name) {
      return name;
    }
    if (like.actor_name) {
      return like.actor_name;
    }
    const handle = like.user?.username?.trim();
    if (handle) {
      return handle;
    }
    return 'User';
  }

  public getLikeAuthorHandle(like: WorkoutLike): string | null {
    return like.user?.username?.trim() || null;
  }

  public async likeWorkout(): Promise<void> {
    const workout = this.workoutState();
    if (!workout || this.isLiking() || workout.liked_by_me) {
      return;
    }

    const previousLiked = workout.liked_by_me;
    const previousCount = workout.likes_count || 0;
    const nextCount = previousCount + 1;

    const currentUser = this.user.getUserInfo()();
    const myLike: WorkoutLike = {
      id: Date.now(),
      user_id: currentUser?.profile?.id,
      user: currentUser?.profile
        ? {
            id: currentUser.profile.id,
            username: currentUser.profile.username,
            name: currentUser.profile.name,
            handle: currentUser.profile.username ? `@${currentUser.profile.username}` : '',
            actor_url: '',
            icon_url: currentUser.profile.icon_url || '',
            is_external: false,
            is_own: true,
            is_following: false,
          }
        : undefined,
      actor_name: currentUser?.name || currentUser?.profile?.name || currentUser?.username,
      avatar_url: currentUser?.profile?.icon_url,
      created_at: new Date().toISOString(),
    };

    // Instant local updates
    this.workoutState.update((current) =>
      current
        ? {
            ...current,
            liked_by_me: true,
            likes_count: nextCount,
          }
        : current,
    );

    this.workoutLikes.update((current) => {
      const exists = current.some(
        (l) =>
          (l.user?.username && currentUser?.username && l.user.username === currentUser.username) ||
          (l.user_id && myLike.user_id && l.user_id === myLike.user_id),
      );
      return exists ? current : [myLike, ...current];
    });

    this.isLiking.set(true);
    try {
      const response = await firstValueFrom(this.api.likeWorkout(workout.id));
      if (response?.results) {
        this.workoutState.update((current) =>
          current
            ? {
                ...current,
                liked_by_me: true,
                likes_count: response.results.likes_count,
              }
            : current,
        );
      }
    } catch (error) {
      console.error('Failed to like workout:', error);
      this.workoutState.update((current) =>
        current
          ? {
              ...current,
              liked_by_me: previousLiked,
              likes_count: previousCount,
            }
          : current,
      );
      this.workoutLikes.update((current) => current.filter((l) => l.id !== myLike.id));
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

  private formatUserHandle(user?: UserSummary): string {
    if (user?.handle) {
      return user.handle.replace(/^@/, '');
    }
    return user?.username?.trim() || '';
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
