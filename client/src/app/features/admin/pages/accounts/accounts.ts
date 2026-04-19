import { ChangeDetectionStrategy, Component, inject, OnInit, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { TranslatePipe } from '@ngx-translate/core';
import { firstValueFrom } from 'rxjs';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { Api } from '../../../../core/services/api';
import { UserProfile } from '../../../../core/types/user';

@Component({
  selector: 'app-admin-accounts',
  imports: [RouterLink, AppIcon, TranslatePipe],
  templateUrl: './accounts.html',
  styleUrl: './accounts.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AdminAccounts implements OnInit {
  private api = inject(Api);

  public readonly users = signal<UserProfile[]>([]);
  public readonly loading = signal(true);
  public readonly error = signal<string | null>(null);
  public readonly deleteConfirm = signal<number | null>(null);

  public ngOnInit(): void {
    this.loadUsers();
  }

  public async loadUsers(): Promise<void> {
    this.loading.set(true);
    this.error.set(null);

    try {
      const usersResponse = await firstValueFrom(this.api.getUsers());
      if (usersResponse?.results) {
        this.users.set(usersResponse.results);
      }
    } catch (err) {
      this.error.set('Failed to load users: ' + (err instanceof Error ? err.message : String(err)));
    } finally {
      this.loading.set(false);
    }
  }

  public confirmDelete(userId: number): void {
    this.deleteConfirm.set(userId);
  }

  public cancelDelete(): void {
    this.deleteConfirm.set(null);
  }

  public async deleteUser(userId: number): Promise<void> {
    this.error.set(null);

    try {
      await firstValueFrom(this.api.deleteUser(userId));
      await this.loadUsers();
      this.deleteConfirm.set(null);
    } catch (err) {
      this.error.set(
        'Failed to delete user: ' + (err instanceof Error ? err.message : String(err)),
      );
      this.deleteConfirm.set(null);
    }
  }
}
