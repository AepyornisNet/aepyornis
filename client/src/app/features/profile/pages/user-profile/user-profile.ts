import { ChangeDetectionStrategy, Component, inject, OnInit, signal } from '@angular/core';
import { DatePipe } from '@angular/common';

import { firstValueFrom } from 'rxjs';
import { Api } from '../../../../core/services/api';
import { Totals, WorkoutRecord } from '../../../../core/types/workout';
import { WorkoutCalendar } from '../../components/workout-calendar/workout-calendar';
import { KeyMetrics } from '../../components/key-metrics/key-metrics';
import { Records } from '../../components/records/records';
import { TranslatePipe, TranslateService } from '@ngx-translate/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { ActivityPubProfileSummary } from '../../../../core/types/user';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';

@Component({
  selector: 'app-user-profile',
  imports: [WorkoutCalendar, KeyMetrics, Records, TranslatePipe, RouterLink, AppIcon, DatePipe],
  templateUrl: './user-profile.html',
  styleUrl: './user-profile.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class UserProfile implements OnInit {
  private api = inject(Api);
  private translate = inject(TranslateService);
  private route = inject(ActivatedRoute);

  public readonly totals = signal<Totals | null>(null);
  public readonly records = signal<WorkoutRecord[]>([]);
  public readonly profileSummary = signal<ActivityPubProfileSummary | null>(null);
  public readonly currentHandle = signal<string | null>(null);
  public readonly followActionBusy = signal(false);
  public readonly loading = signal(true);
  public readonly error = signal<string | null>(null);
  public readonly successMessage = signal<string | null>(null);

  public ngOnInit(): void {
    this.route.paramMap.subscribe((params) => {
      const handle = params.get('handle');
      this.currentHandle.set(handle);
      this.loadProfileData(handle);
    });
  }

  public async loadProfileData(handle: string | null): Promise<void> {
    this.loading.set(true);
    this.error.set(null);
    this.successMessage.set(null);

    try {
      const profileResponse = await firstValueFrom(this.api.getUserProfileSummary(handle || undefined));
      if (!profileResponse?.results) {
        return;
      }

      this.profileSummary.set(profileResponse.results);

      if (profileResponse.results.is_external) {
        this.totals.set(null);
        this.records.set([]);
        return;
      }

      if (profileResponse.results.username) {
        try {
          const actor = await firstValueFrom(
            this.api.getLocalActivityPubActor(profileResponse.results.username),
          );

          this.profileSummary.update((current) =>
            current
              ? {
                  ...current,
                  member_since: actor?.published || current.member_since,
                  icon_url: this.extractActorIcon(actor) || current.icon_url,
                }
              : current,
          );
        } catch {
          this.error.set(
            this.translate.instant('Failed to load {{page}} data.', {
              page: this.translate.instant('profile'),
            }),
          );
        }
      }

      const [totalsResponse, recordsResponse] = await Promise.all([
        firstValueFrom(this.api.getTotals(handle || undefined)),
        firstValueFrom(this.api.getRecords(handle || undefined)),
      ]);

      if (totalsResponse) {
        this.totals.set(totalsResponse.results);
      }

      if (recordsResponse) {
        this.records.set(recordsResponse.results);
      }
    } catch (err) {
      console.error('Failed to load profile data:', err);
      this.error.set(
        this.translate.instant('Failed to load {{page}} data.', {
          page: this.translate.instant('profile'),
        }),
      );
    } finally {
      this.loading.set(false);
    }
  }

  public async toggleFollow(): Promise<void> {
    const summary = this.profileSummary();
    if (!summary || summary.is_own || this.followActionBusy()) {
      return;
    }

    const wasFollowing = summary.is_following;

    this.followActionBusy.set(true);
    this.error.set(null);
    this.successMessage.set(null);

    try {
      const response = summary.is_following
        ? await firstValueFrom(this.api.unfollowUserByHandle(summary.handle))
        : await firstValueFrom(this.api.followUserByHandle(summary.handle));

      if (response?.results) {
        this.profileSummary.set({
          ...summary,
          is_following: response.results.is_following,
          followers_count: response.results.followers_count,
          following_count: response.results.following_count,
        });

        this.successMessage.set(
          this.translate.instant(wasFollowing ? 'Unfollowed user' : 'Followed user'),
        );
      }
    } catch (err) {
      console.error('Failed to toggle follow:', err);
      this.error.set(this.translate.instant('Failed to update follow state. Please try again.'));
    } finally {
      this.followActionBusy.set(false);
    }
  }

  public copyToClipboard(text: string): void {
    navigator.clipboard
      .writeText(text)
      .then(() => {
        this.successMessage.set(this.translate.instant('Copied to clipboard'));
      })
      .catch((err) => {
        console.error('Failed to copy to clipboard:', err);
        this.error.set(this.translate.instant('Failed to copy to clipboard'));
      });
  }

  private extractActorIcon(actor: unknown): string {
    if (!actor || typeof actor !== 'object') {
      return '';
    }

    const iconValue = (actor as { icon?: unknown }).icon;
    if (!iconValue) {
      return '';
    }

    if (typeof iconValue === 'string') {
      return iconValue;
    }

    if (typeof iconValue === 'object') {
      const iconObject = iconValue as { url?: unknown };
      if (typeof iconObject.url === 'string') {
        return iconObject.url;
      }
    }

    return '';
  }
}
