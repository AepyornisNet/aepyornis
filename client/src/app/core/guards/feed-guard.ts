import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { User } from '../services/user';

export const feedGuard: CanActivateFn = (route, state) => {
  const userService = inject(User);
  const router = inject(Router);

  const userInfo = userService.getUserInfo()();

  if (!userInfo?.isAuthenticated) {
    router.navigate(['/login'], { queryParams: { returnUrl: state.url } });
    return false;
  }

  if (!userInfo.profile?.activity_pub) {
    router.navigate(['/profile']);
    return false;
  }

  return true;
};
