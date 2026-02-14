import { ChangeDetectionStrategy, Component, effect, inject, OnInit, signal } from '@angular/core';

import { _, TranslatePipe } from '@ngx-translate/core';
import {
  FormBuilder,
  FormGroup,
  ReactiveFormsModule,
  ValidationErrors,
  Validators,
} from '@angular/forms';
import { HttpErrorResponse } from '@angular/common/http';
import { ActivatedRoute, Router } from '@angular/router';
import { User } from '../../../../core/services/user';
import { AppConfig } from '../../../../core/services/app-config';
import { Api } from '../../../../core/services/api';
import { PublicLayout } from '../../../../layouts/public-layout/public-layout';

@Component({
  selector: 'app-login',
  imports: [ReactiveFormsModule, PublicLayout, TranslatePipe],
  templateUrl: './login.html',
  styleUrl: './login.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Login implements OnInit {
  private userService = inject(User);
  private appConfig = inject(AppConfig);
  private api = inject(Api);
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private fb = inject(FormBuilder);

  // Login form (reactive form)
  public loginForm!: FormGroup;
  public readonly errorMessage = signal<string | null>(null);
  public readonly loginSubmitting = signal(false);
  public readonly returnUrl = signal('/dashboard');

  // Register form (reactive form with 3 fields)
  public registerForm!: FormGroup;
  public readonly registerErrorMessage = signal<string | null>(null);
  public readonly registerSuccessMessage = signal<string | null>(null);
  public readonly registerSubmitting = signal(false);

  public get isRegistrationDisabled(): boolean {
    return this.appConfig.isRegistrationDisabled();
  }

  public constructor() {
    // Monitor auth state changes and redirect when authenticated
    effect(() => {
      if (this.userService.isAuthenticated() && !this.userService.isCheckingAuth()) {
        this.router.navigate([this.returnUrl()]);
      }
    });
  }

  public ngOnInit(): void {
    // Initialize login form
    this.loginForm = this.fb.group({
      username: ['', Validators.required],
      password: ['', Validators.required],
    });

    // Initialize register form with custom validator for password matching
    this.registerForm = this.fb.group(
      {
        username: ['', Validators.required],
        password: ['', [Validators.required, Validators.minLength(6)]],
        confirmPassword: ['', Validators.required],
      },
      {
        validators: this.passwordMatchValidator,
      },
    );

    // Check if there's an error parameter in the URL
    this.route.queryParams.subscribe((params) => {
      if (params['error']) {
        this.errorMessage.set(decodeURIComponent(params['error']));
      }
      if (params['returnUrl']) {
        this.returnUrl.set(params['returnUrl']);
      }
    });
  }

  // Custom validator to ensure passwords match
  private passwordMatchValidator(group: FormGroup): ValidationErrors | null {
    const password = group.get('password')?.value;
    const confirmPassword = group.get('confirmPassword')?.value;

    // Only validate if both fields have values
    if (!password || !confirmPassword) {
      return null;
    }

    return password === confirmPassword ? null : { passwordMismatch: true };
  }

  public onSubmit(): void {
    if (this.loginForm.invalid || this.loginSubmitting()) {
      return;
    }

    // Clear any previous errors
    this.errorMessage.set(null);

    this.loginSubmitting.set(true);

    const formValue = this.loginForm.value;

    this.api
      .signIn({
        username: String(formValue.username ?? ''),
        password: String(formValue.password ?? ''),
      })
      .subscribe({
        next: (response) => {
          if (response?.results) {
            this.userService.setAuthenticatedUser(response.results);
            void this.router.navigate([this.returnUrl()]);
            return;
          }

          this.errorMessage.set(_('Login failed'));
        },
        error: (err: HttpErrorResponse) => {
          const apiMessage = err.error?.errors?.[0];
          this.errorMessage.set(apiMessage || _('Login failed'));
          this.loginSubmitting.set(false);
        },
        complete: () => {
          this.loginSubmitting.set(false);
        },
      });
  }

  public onRegister(): void {
    if (this.registerForm.invalid || this.registerSubmitting()) {
      return;
    }

    // Clear any previous messages
    this.registerErrorMessage.set(null);
    this.registerSuccessMessage.set(null);

    this.registerSubmitting.set(true);

    const formValue = this.registerForm.value;

    this.api
      .register({
        username: String(formValue.username ?? ''),
        password: String(formValue.password ?? ''),
        name: String(formValue.username ?? ''),
      })
      .subscribe({
        next: (response) => {
          this.registerSuccessMessage.set(
            response?.results?.message ?? 'Your account has been created',
          );
          this.registerForm.reset();
        },
        error: (err: HttpErrorResponse) => {
          const apiMessage = err.error?.errors?.[0];
          this.registerErrorMessage.set(apiMessage || 'Registration failed');
          this.registerSubmitting.set(false);
        },
        complete: () => {
          this.registerSubmitting.set(false);
        },
      });
  }
}
