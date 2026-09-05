import { ChangeDetectionStrategy, Component, inject, OnInit, signal } from '@angular/core';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { disabled, email, form, FormField, FormRoot, required } from '@angular/forms/signals';
import { Api } from '../../../../core/services/api';
import { UserProfile } from '../../../../core/types/user';
import { TranslatePipe, TranslateService } from '@ngx-translate/core';

@Component({
  selector: 'app-user-edit',
  imports: [RouterLink, AppIcon, FormField, FormRoot, TranslatePipe],
  templateUrl: './user-edit.html',
  styleUrl: './user-edit.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class UserEdit implements OnInit {
  private api = inject(Api);
  private router = inject(Router);
  private route = inject(ActivatedRoute);
  private translate = inject(TranslateService);

  public readonly userId = signal<number>(0);
  public readonly user = signal<UserProfile | null>(null);
  public readonly loading = signal(true);
  public readonly saving = signal(false);
  public readonly error = signal<string | null>(null);

  public readonly userModel = signal({
    email: '',
    username: '',
    name: '',
    password: '',
    active: false,
    admin: false,
  });

  public readonly userForm = form(
    this.userModel,
    (s) => {
      required(s.email);
      email(s.email);
      required(s.name);
      disabled(s.email, { when: () => this.saving() });
      disabled(s.username, { when: () => this.saving() });
      disabled(s.name, { when: () => this.saving() });
      disabled(s.password, { when: () => this.saving() });
      disabled(s.active, { when: () => this.saving() });
      disabled(s.admin, { when: () => this.saving() });
    },
    {
      submission: {
        action: () => this.saveUser(),
      },
    },
  );

  public ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('id');
    if (id) {
      this.userId.set(parseInt(id, 10));
      this.loadUser();
    } else {
      this.error.set(this.translate.instant('Invalid user ID'));
      this.loading.set(false);
    }
  }

  public async loadUser(): Promise<void> {
    this.loading.set(true);
    this.error.set(null);

    try {
      const response = await firstValueFrom(this.api.getUser(this.userId()));
      if (response?.results) {
        this.user.set(response.results);
        this.userModel.set({
          email: response.results.email || '',
          username: response.results.username || '',
          name: response.results.name || '',
          password: '',
          active: Boolean(response.results.active),
          admin: Boolean(response.results.admin),
        });
      }
    } catch (err) {
      this.error.set(
        this.translate.instant('Failed to load user: {{message}}', {
          message: err instanceof Error ? err.message : String(err),
        }),
      );
    } finally {
      this.loading.set(false);
    }
  }

  public async saveUser(): Promise<void> {
    if (this.userForm().invalid()) {
      return;
    }

    this.saving.set(true);
    this.error.set(null);

    try {
      const formValue = this.userModel();
      const updateData = {
        email: formValue.email,
        ...(formValue.username && { username: formValue.username }),
        name: formValue.name,
        active: formValue.active,
        admin: formValue.admin,
        ...(formValue.password && { password: formValue.password }),
      };

      const response = await firstValueFrom(this.api.updateUser(this.userId(), updateData));
      if (response?.results) {
        this.user.set(response.results);
        this.userModel.update((m) => ({ ...m, password: '' }));
        this.router.navigate(['/admin/accounts']);
      }
    } catch (err) {
      this.error.set(
        this.translate.instant('Failed to save user: {{message}}', {
          message: err instanceof Error ? err.message : String(err),
        }),
      );
    } finally {
      this.saving.set(false);
    }
  }
}
