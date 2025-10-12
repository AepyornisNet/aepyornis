import { Component, OnInit, signal, inject, ChangeDetectionStrategy, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { Api } from '../../../core/services/api';
import { AppIcon } from '../app-icon/app-icon';
import { UserProfile } from '../../types/user';

@Component({
  selector: 'app-other-users',
  imports: [CommonModule, RouterLink, AppIcon],
  templateUrl: './other-users.html',
  styleUrl: './other-users.scss',
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class OtherUsers implements OnInit {
  private api = inject(Api);
  
  users = signal<UserProfile[]>([]);
  currentUserId = signal<number | null>(null);
  loading = signal(true);
  error = signal<string | null>(null);
  
  // Filter out the current user
  otherUsers = computed(() => {
    const userId = this.currentUserId();
    if (userId === null) {
      return this.users();
    }
    return this.users().filter(u => u.id !== userId);
  });

  ngOnInit() {
    this.loadUsers();
  }

  private async loadUsers() {
    this.loading.set(true);
    this.error.set(null);
    
    try {
      // Load current user and all users in parallel
      const [whoamiResponse, usersResponse] = await Promise.all([
        firstValueFrom(this.api.whoami()),
        firstValueFrom(this.api.getUsers())
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
