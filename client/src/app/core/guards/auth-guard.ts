import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { User } from '../services/user';

export const authGuard: CanActivateFn = (route, state) => {
  const userService = inject(User);
  const router = inject(Router);

  if (userService.isAuthenticated()) {
    return true;
  }

  // Redirect to login with the return URL
  router.navigate(['/login'], { queryParams: { returnUrl: state.url } });
  return false;
};
