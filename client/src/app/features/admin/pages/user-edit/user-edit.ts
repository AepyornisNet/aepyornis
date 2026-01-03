import { ChangeDetectionStrategy, Component, inject, OnInit, signal } from '@angular/core';

import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { FormBuilder, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { Api } from '../../../../core/services/api';
import { UserProfile } from '../../../../core/types/user';
import { TranslatePipe } from '@ngx-translate/core';

@Component({
  selector: 'app-user-edit',
  imports: [RouterLink, AppIcon, ReactiveFormsModule, TranslatePipe],
  templateUrl: './user-edit.html',
  styleUrl: './user-edit.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class UserEdit implements OnInit {
  private api = inject(Api);
  private router = inject(Router);
  private route = inject(ActivatedRoute);
  private fb = inject(FormBuilder);

  public readonly userId = signal<number>(0);
  public readonly user = signal<UserProfile | null>(null);
  public readonly loading = signal(true);
  public readonly saving = signal(false);
  public readonly error = signal<string | null>(null);

  // Reactive form
  public userForm!: FormGroup;

  public ngOnInit(): void {
    // Initialize form
    this.userForm = this.fb.group({
      username: ['', Validators.required],
      name: ['', Validators.required],
      password: [''],
      active: [false],
      admin: [false],
    });

    const id = this.route.snapshot.paramMap.get('id');
    if (id) {
      this.userId.set(parseInt(id, 10));
      this.loadUser();
    } else {
      this.error.set('Invalid user ID');
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
        // Populate form with loaded data
        this.userForm.patchValue({
          username: response.results.username,
          name: response.results.name,
          password: '', // Don't populate password
          active: response.results.active,
          admin: response.results.admin,
        });
      }
    } catch (err) {
      this.error.set('Failed to load user: ' + (err instanceof Error ? err.message : String(err)));
    } finally {
      this.loading.set(false);
    }
  }

  public async saveUser(): Promise<void> {
    if (this.userForm.invalid) {
      return;
    }

    this.saving.set(true);
    this.error.set(null);

    try {
      const formValue = this.userForm.value;
      const updateData = {
        username: formValue.username,
        name: formValue.name,
        active: formValue.active,
        admin: formValue.admin,
        ...(formValue.password && { password: formValue.password }),
      };

      const response = await firstValueFrom(this.api.updateUser(this.userId(), updateData));
      if (response?.results) {
        this.user.set(response.results);
        // Clear password field after successful update
        this.userForm.patchValue({ password: '' });
        // Navigate back to admin page
        this.router.navigate(['/admin']);
      }
    } catch (err) {
      this.error.set('Failed to save user: ' + (err instanceof Error ? err.message : String(err)));
    } finally {
      this.saving.set(false);
    }
  }
}
