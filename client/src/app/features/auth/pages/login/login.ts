import { ChangeDetectionStrategy, Component, effect, inject, OnInit, signal } from '@angular/core';
import { _, TranslatePipe, TranslateService } from '@ngx-translate/core';
import {
  email,
  form,
  FormField,
  FormRoot,
  minLength,
  required,
  validate,
} from '@angular/forms/signals';
import { HttpErrorResponse } from '@angular/common/http';
import { ActivatedRoute, Router } from '@angular/router';
import { User } from '../../../../core/services/user';
import { AppConfig } from '../../../../core/services/app-config';
import { Api } from '../../../../core/services/api';
import { PublicLayout } from '../../../../layouts/public-layout/public-layout';

@Component({
  selector: 'app-login',
  imports: [FormField, FormRoot, PublicLayout, TranslatePipe],
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
  private translate = inject(TranslateService);

  // Login form
  public readonly loginModel = signal({
    email: '',
    password: '',
  });

  public readonly loginSignalForm = form(
    this.loginModel,
    (s) => {
      required(s.email);
      email(s.email);
      required(s.password);
    },
    {
      submission: {
        action: async () => {
          this.onSubmit();
        },
      },
    },
  );

  public readonly errorMessage = signal<string | null>(null);
  public readonly loginSubmitting = signal(false);
  public readonly returnUrl = signal('/feed');

  // Register form
  public readonly registerModel = signal({
    username: '',
    email: '',
    password: '',
    confirmPassword: '',
  });

  public readonly registerSignalForm = form(
    this.registerModel,
    (s) => {
      required(s.username);
      required(s.email);
      email(s.email);
      required(s.password);
      minLength(s.password, 6);
      required(s.confirmPassword);
      validate(s.confirmPassword, ({ valueOf, value }) => {
        if (!value() || !valueOf(s.password)) {
          return undefined;
        }
        return value() === valueOf(s.password)
          ? undefined
          : {
              kind: 'passwordMismatch',
              message: this.translate.instant('Passwords do not match'),
            };
      });
    },
    {
      submission: {
        action: async () => {
          this.onRegister();
        },
      },
    },
  );

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

  public onSubmit(): void {
    if (this.loginSignalForm().invalid() || this.loginSubmitting()) {
      return;
    }

    // Clear any previous errors
    this.errorMessage.set(null);

    this.loginSubmitting.set(true);

    const formValue = this.loginModel();

    this.api
      .signIn({
        email: String(formValue.email ?? ''),
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
    if (this.registerSignalForm().invalid() || this.registerSubmitting()) {
      return;
    }

    // Clear any previous messages
    this.registerErrorMessage.set(null);
    this.registerSuccessMessage.set(null);

    this.registerSubmitting.set(true);

    const formValue = this.registerModel();
    const currentLanguage = localStorage.getItem('locale') || 'browser';

    this.api
      .register({
        email: String(formValue.email ?? ''),
        username: String(formValue.username ?? ''),
        password: String(formValue.password ?? ''),
        name: String(formValue.username ?? ''),
        language: currentLanguage,
      })
      .subscribe({
        next: (response) => {
          this.registerSuccessMessage.set(
            response?.results?.message ?? _('Your account has been created'),
          );
          this.registerModel.set({
            username: '',
            email: '',
            password: '',
            confirmPassword: '',
          });
        },
        error: (err: HttpErrorResponse) => {
          const apiMessage = err.error?.errors?.[0];
          this.registerErrorMessage.set(apiMessage || _('Registration failed'));
          this.registerSubmitting.set(false);
        },
        complete: () => {
          this.registerSubmitting.set(false);
        },
      });
  }
}
