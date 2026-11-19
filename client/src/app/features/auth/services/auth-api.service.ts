import { inject, Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { APIResponse } from '../../../core/types/api-response';
import { RegisterRequest, SignInRequest } from '../../../core/types/auth';
import { UserProfile } from '../../../core/types/user';

@Injectable({
  providedIn: 'root',
})
export class AuthApiService {
  private http = inject(HttpClient);
  private baseUrl = '/api/v2';

  public signIn(payload: SignInRequest): Observable<APIResponse<UserProfile>> {
    return this.http.post<APIResponse<UserProfile>>(`${this.baseUrl}/auth/signin`, payload);
  }

  public register(payload: RegisterRequest): Observable<APIResponse<{ message: string }>> {
    return this.http.post<APIResponse<{ message: string }>>(
      `${this.baseUrl}/auth/register`,
      payload,
    );
  }

  public signOut(): Observable<APIResponse<{ message: string }>> {
    return this.http.post<APIResponse<{ message: string }>>(`${this.baseUrl}/auth/signout`, {});
  }

  public whoami(): Observable<APIResponse<UserProfile>> {
    return this.http.get<APIResponse<UserProfile>>(`${this.baseUrl}/whoami`);
  }
}
