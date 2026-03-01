import {
  ChangeDetectionStrategy,
  Component,
  computed,
  effect,
  inject,
  input,
  signal,
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { TranslatePipe } from '@ngx-translate/core';
import { firstValueFrom } from 'rxjs';
import { WorkoutReply } from '../../../../core/types/workout';
import { Api } from '../../../../core/services/api';

@Component({
  selector: 'app-workout-comments',
  standalone: true,
  imports: [CommonModule, FormsModule, TranslatePipe],
  templateUrl: './workout-comments.html',
  styleUrl: './workout-comments.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class WorkoutCommentsComponent {
  public readonly workoutId = input.required<number>();

  private api = inject(Api);

  public readonly replies = signal<WorkoutReply[]>([]);
  public readonly newComment = signal('');
  public readonly page = signal(1);
  public readonly totalCount = signal(0);
  public readonly hasMore = signal(false);
  public readonly loadingReplies = signal(false);
  public readonly loadingMore = signal(false);
  public readonly isSubmitting = signal(false);
  public readonly canSubmit = computed(
    () => this.newComment().trim().length > 0 && !this.isSubmitting(),
  );
  private readonly perPage = 20;

  public constructor() {
    effect(() => {
      const workoutId = this.workoutId();
      if (workoutId > 0) {
        void this.loadReplies(true);
      }
    });
  }

  public setNewComment(value: string): void {
    this.newComment.set(value);
  }

  public getAuthorName(reply: WorkoutReply): string {
    if (reply.user) {
      return reply.user.name;
    }
    if (reply.actor_name) {
      return reply.actor_name;
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

  public isRemoteComment(reply: WorkoutReply): boolean {
    return !!reply.actor_iri;
  }

  public hasAvatarUrl(reply: WorkoutReply): boolean {
    return !!(reply.avatar_url && reply.avatar_url.trim());
  }

  public getInitial(reply: WorkoutReply): string {
    const name = this.getAuthorName(reply);
    return (name.charAt(0) || '?').toUpperCase();
  }

  public async loadMore(): Promise<void> {
    if (!this.hasMore() || this.loadingMore() || this.loadingReplies()) {
      return;
    }

    await this.loadReplies(false);
  }

  public async submitComment(): Promise<void> {
    const comment = this.newComment().trim();
    if (!comment || this.isSubmitting()) {
      return;
    }

    this.isSubmitting.set(true);
    try {
      const response = await firstValueFrom(this.api.createReply(this.workoutId(), comment));
      if (response?.results) {
        this.replies.update((currentReplies) => [response.results, ...currentReplies]);
        this.totalCount.update((count) => count + 1);
        this.newComment.set('');
      }
    } catch (error) {
      console.error('Failed to create reply:', error);
    } finally {
      this.isSubmitting.set(false);
    }
  }

  private async loadReplies(reset: boolean): Promise<void> {
    const workoutId = this.workoutId();
    if (!workoutId) {
      return;
    }

    if (reset) {
      this.loadingReplies.set(true);
      this.page.set(1);
    } else {
      this.loadingMore.set(true);
    }

    try {
      const targetPage = reset ? 1 : this.page() + 1;
      const response = await firstValueFrom(
        this.api.getWorkoutReplies(workoutId, { page: targetPage, per_page: this.perPage }),
      );

      if (reset) {
        this.replies.set(response?.results || []);
      } else {
        this.replies.update((currentReplies) => [...currentReplies, ...(response?.results || [])]);
      }

      this.page.set(response?.page || targetPage);
      this.totalCount.set(response?.total_count || this.replies().length);
      this.hasMore.set((response?.page || 1) < (response?.total_pages || 1));
    } catch (error) {
      console.error('Failed to load workout replies:', error);
      if (reset) {
        this.replies.set([]);
      }
      this.hasMore.set(false);
    } finally {
      this.loadingReplies.set(false);
      this.loadingMore.set(false);
    }
  }
}
