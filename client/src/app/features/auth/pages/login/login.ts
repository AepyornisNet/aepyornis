import { Component, OnInit, signal, effect, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormBuilder, FormGroup, FormsModule, ReactiveFormsModule, ValidationErrors, Validators } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { User } from '../../../../core/services/user';
import { AppConfig } from '../../../../core/services/app-config';
import { AUTH_LOGIN_URL, AUTH_REGISTER_URL } from '../../../../core/types/auth';
import { PublicLayout } from '../../../../layouts/public-layout/public-layout';

@Component({
  selector: 'app-login',
  imports: [CommonModule, FormsModule, ReactiveFormsModule, PublicLayout],
  templateUrl: './login.html',
  styleUrl: './login.scss'
})
export class Login implements OnInit {
  private userService = inject(User);
  private appConfig = inject(AppConfig);
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private fb = inject(FormBuilder);

  // Login form (reactive form)
  loginForm!: FormGroup;
  errorMessage = signal<string | null>(null);
  returnUrl = signal('/dashboard');

  // Register form (reactive form with 3 fields)
  registerForm!: FormGroup;
  registerErrorMessage = signal<string | null>(null);
  registerSuccessMessage = signal<string | null>(null);

  get isRegistrationDisabled() {
    return this.appConfig.isRegistrationDisabled();
  }

  constructor() {
    // Monitor auth state changes and redirect when authenticated
    effect(() => {
      if (this.userService.isAuthenticated() && !this.userService.isCheckingAuth()) {
        this.router.navigate([this.returnUrl()]);
      }
    });
  }

  ngOnInit() {
    // Initialize login form
    this.loginForm = this.fb.group({
      username: ['', Validators.required],
      password: ['', Validators.required]
    });

    // Initialize register form with custom validator for password matching
    this.registerForm = this.fb.group({
      username: ['', Validators.required],
      password: ['', [Validators.required, Validators.minLength(6)]],
      confirmPassword: ['', Validators.required]
    }, {
      validators: this.passwordMatchValidator
    });

    // Check if there's an error parameter in the URL
    this.route.queryParams.subscribe(params => {
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

  onSubmit() {
    if (this.loginForm.invalid) {
      return;
    }

    // Clear any previous errors
    this.errorMessage.set(null);

    const formValue = this.loginForm.value;

    // Create and submit a form to the backend
    const form = document.createElement('form');
    form.method = 'POST';
    form.action = AUTH_LOGIN_URL;

    const usernameInput = document.createElement('input');
    usernameInput.type = 'hidden';
    usernameInput.name = 'username';
    usernameInput.value = formValue.username;
    form.appendChild(usernameInput);

    const passwordInput = document.createElement('input');
    passwordInput.type = 'hidden';
    passwordInput.name = 'password';
    passwordInput.value = formValue.password;
    form.appendChild(passwordInput);

    document.body.appendChild(form);
    form.submit();
  }

  onRegister() {
    if (this.registerForm.invalid) {
      return;
    }

    // Clear any previous messages
    this.registerErrorMessage.set(null);
    this.registerSuccessMessage.set(null);

    const formValue = this.registerForm.value;

    // Create and submit a form to the backend
    const form = document.createElement('form');
    form.method = 'POST';
    form.action = AUTH_REGISTER_URL;

    const usernameInput = document.createElement('input');
    usernameInput.type = 'hidden';
    usernameInput.name = 'username';
    usernameInput.value = formValue.username;
    form.appendChild(usernameInput);

    const passwordInput = document.createElement('input');
    passwordInput.type = 'hidden';
    passwordInput.name = 'password';
    passwordInput.value = formValue.password;
    form.appendChild(passwordInput);

    document.body.appendChild(form);
    form.submit();
  }
}
