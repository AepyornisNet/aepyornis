import { ChangeDetectionStrategy, Component, inject, OnInit, signal } from '@angular/core';
import { form, FormField, FormRoot } from '@angular/forms/signals';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { TranslatePipe, TranslateService } from '@ngx-translate/core';
import { Api } from '../../../../core/services/api';
import { ActivityPubActor, ActivityPubProfileSummary } from '../../../../core/types/user';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { Avatar } from '../../../../core/components/avatar/avatar';

@Component({
  selector: 'app-profile-search-page',
  imports: [FormField, FormRoot, RouterLink, TranslatePipe, AppIcon, Avatar],
  templateUrl: './search.html',
  styleUrl: './search.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ProfileSearchPage implements OnInit {
  private api = inject(Api);
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private translate = inject(TranslateService);

  public readonly searchModel = signal({
    q: '',
  });
  public readonly searchSignalForm = form(this.searchModel, () => undefined, {
    submission: {
      action: async () => {
        this.onSearch();
      },
    },
  });
  public readonly results = signal<ActivityPubProfileSummary[]>([]);
  public readonly loading = signal(false);
  public readonly searched = signal(false);
  public readonly error = signal<string | null>(null);
  public readonly successMessage = signal<string | null>(null);
  public readonly followActionBusy = signal<Record<string, boolean>>({});

  public ngOnInit(): void {
    this.route.queryParamMap.subscribe((params) => {
      const query = params.get('q')?.trim() ?? '';
      this.searchModel.set({ q: query });
      this.error.set(null);
      this.successMessage.set(null);

      if (query) {
        void this.loadResults(query);
        return;
      }

      this.results.set([]);
      this.searched.set(false);
      this.loading.set(false);
    });
  }

  public onSearch(): void {
    const query = this.searchModel().q.trim();
    void this.router.navigate([], {
      relativeTo: this.route,
      queryParams: { q: query || null },
    });
  }

  public isBusy(handle: string): boolean {
    return !!this.followActionBusy()[handle];
  }

  public async toggleFollow(profile: ActivityPubProfileSummary): Promise<void> {
    if (this.isBusy(profile.handle)) {
      return;
    }

    const wasFollowing = profile.is_following;

    this.followActionBusy.update((value) => ({ ...value, [profile.handle]: true }));
    this.error.set(null);
    this.successMessage.set(null);

    try {
      const response = wasFollowing
        ? await firstValueFrom(this.api.unfollowUserByHandle(profile.handle))
        : await firstValueFrom(this.api.followUserByHandle(profile.handle));

      if (response?.results) {
        this.results.update((results) =>
          results.map((item) =>
            item.handle === profile.handle
              ? {
                  ...item,
                  is_following: response.results.is_following,
                  followers_count: response.results.followers_count,
                  following_count: response.results.following_count,
                }
              : item,
          ),
        );

        this.successMessage.set(
          this.translate.instant(wasFollowing ? 'Unfollowed user' : 'Followed user'),
        );
      }
    } catch {
      this.error.set(this.translate.instant('Failed to update follow state. Please try again.'));
    } finally {
      this.followActionBusy.update((value) => ({ ...value, [profile.handle]: false }));
    }
  }

  private async loadResults(query: string): Promise<void> {
    this.loading.set(true);
    this.searched.set(true);
    this.error.set(null);
    this.successMessage.set(null);

    try {
      const response = await firstValueFrom(this.api.searchProfiles(query));
      const results = response?.results ?? [];
      this.results.set(await this.enrichLocalResults(results));
    } catch {
      this.results.set([]);
      this.error.set(this.translate.instant('Failed to load {{page}} data.', { page: 'profiles' }));
    } finally {
      this.loading.set(false);
    }
  }

  private async enrichLocalResults(
    results: ActivityPubProfileSummary[],
  ): Promise<ActivityPubProfileSummary[]> {
    return Promise.all(
      results.map(async (profile) => {
        if (profile.is_external || !profile.username) {
          return profile;
        }

        const actor = await firstValueFrom(
          this.api.getLocalActivityPubActor(profile.username),
        ).catch(() => null);
        const iconURL = this.extractActorIcon(actor);

        return iconURL ? { ...profile, icon_url: iconURL } : profile;
      }),
    );
  }

  private extractActorIcon(actor: ActivityPubActor | null): string {
    if (!actor?.icon) {
      return '';
    }

    if (typeof actor.icon === 'string') {
      return actor.icon;
    }

    return typeof actor.icon.url === 'string' ? actor.icon.url : '';
  }
}
