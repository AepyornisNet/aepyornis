import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  OnInit,
  signal,
} from '@angular/core';

import { TranslatePipe } from '@ngx-translate/core';
import { RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { Api } from '../../../core/services/api';
import { AppIcon } from '../app-icon/app-icon';
import { UserProfile } from '../../types/user';

@Component({
  selector: 'app-other-users',
  imports: [RouterLink, AppIcon, TranslatePipe],
  templateUrl: './other-users.html',
  styleUrl: './other-users.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class OtherUsers implements OnInit {
  private api = inject(Api);

  public readonly users = signal<UserProfile[]>([]);
  public readonly currentUserId = signal<number | null>(null);
  public readonly loading = signal(true);
  public readonly error = signal<string | null>(null);

  // Filter out the current user
  public readonly otherUsers = computed(() => {
    const userId = this.currentUserId();
    if (userId === null) {
      return this.users();
    }
    return this.users().filter((u) => u.id !== userId);
  });

  public ngOnInit(): void {
    this.loadUsers();
  }

  private async loadUsers(): Promise<void> {
    this.loading.set(true);
    this.error.set(null);

    try {
      // Load current user and all users in parallel
      const [whoamiResponse, usersResponse] = await Promise.all([
        firstValueFrom(this.api.whoami()),
        firstValueFrom(this.api.getUsers()),
      ]);

      if (whoamiResponse) {
        this.currentUserId.set(whoamiResponse.results.id);
      }

      if (usersResponse) {
        this.users.set(usersResponse.results);
      }
    } catch (err: unknown) {
      console.error('Failed to load users:', err);
      // Check if it's a 403/401 error (not admin) - hide the error in this case
      if (err && typeof err === 'object' && 'status' in err) {
        const status = (err as { status?: number }).status;
        if (status === 403 || status === 401) {
          // User is not admin, just hide the component
          this.users.set([]);
        } else {
          this.error.set('Failed to load users');
        }
      } else {
        this.error.set('Failed to load users');
      }
    } finally {
      this.loading.set(false);
    }
  }
}
