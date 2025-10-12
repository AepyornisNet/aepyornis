import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { User } from '../services/user';

export const adminGuard: CanActivateFn = (route, state) => {
  const userService = inject(User);
  const router = inject(Router);

  const userInfo = userService.getUserInfo()();

  if (!userInfo?.isAuthenticated) {
    // Redirect to login with the return URL
    router.navigate(['/login'], { queryParams: { returnUrl: state.url } });
    return false;
  }

  if (!userInfo?.profile?.admin) {
    // User is authenticated but not an admin, redirect to dashboard
    router.navigate(['/dashboard']);
    return false;
  }

  return true;
};
