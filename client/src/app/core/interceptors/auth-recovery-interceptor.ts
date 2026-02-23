import { HttpErrorResponse, HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { catchError, throwError } from 'rxjs';
import { User } from '../services/user';

const WHOAMI_PATH = '/api/v2/whoami';

export const authRecoveryInterceptor: HttpInterceptorFn = (req, next) => {
  const user = inject(User);

  return next(req).pipe(
    catchError((error: unknown) => {
      if (
        error instanceof HttpErrorResponse &&
        error.status === 401 &&
        !req.url.includes(WHOAMI_PATH)
      ) {
        user.revalidateAfterUnauthorized();
      }

      return throwError(() => error);
    }),
  );
};
