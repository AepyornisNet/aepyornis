import { inject, Injectable } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';
import { APIResponse, PaginatedAPIResponse, PaginationParams } from '../types/api-response';
import {
  Notification,
  UserNotificationSettings,
  UserNotificationSettingsUpdate,
} from '../types/notification';
import { UserWebpushSubscription } from '../types/user';

@Injectable({
  providedIn: 'root',
})
export class NotificationApiService {
  private http = inject(HttpClient);
  private baseUrl = '/api/v2';

  public getNotifications(
    params?: PaginationParams,
  ): Observable<PaginatedAPIResponse<Notification>> {
    let httpParams = new HttpParams();
    if (params?.page) {
      httpParams = httpParams.set('page', params.page.toString());
    }
    if (params?.per_page) {
      httpParams = httpParams.set('per_page', params.per_page.toString());
    }
    return this.http.get<PaginatedAPIResponse<Notification>>(`${this.baseUrl}/notifications`, {
      params: httpParams,
    });
  }

  public getNotificationSettings(): Observable<APIResponse<UserNotificationSettings[]>> {
    return this.http.get<APIResponse<UserNotificationSettings[]>>(
      `${this.baseUrl}/notifications/settings`,
    );
  }

  public updateNotificationSettings(
    method: string,
    payload: UserNotificationSettingsUpdate,
  ): Observable<APIResponse<UserNotificationSettings>> {
    return this.http.post<APIResponse<UserNotificationSettings>>(
      `${this.baseUrl}/notifications/${method}`,
      payload,
    );
  }

  public getWebpushSubscriptions(): Observable<APIResponse<UserWebpushSubscription[]>> {
    return this.http.get<APIResponse<UserWebpushSubscription[]>>(
      `${this.baseUrl}/notifications/webpush/subscriptions`,
    );
  }

  public subscribeWebpush(payload: {
    endpoint: string;
    keys: { auth: string; p256dh: string };
    user_agent?: string;
  }): Observable<APIResponse<UserWebpushSubscription>> {
    return this.http.post<APIResponse<UserWebpushSubscription>>(
      `${this.baseUrl}/notifications/webpush/subscribe`,
      payload,
    );
  }

  public unsubscribeWebpush(endpoint?: string): Observable<APIResponse<{ success: boolean }>> {
    return this.http.post<APIResponse<{ success: boolean }>>(
      `${this.baseUrl}/notifications/webpush/unsubscribe`,
      { endpoint: endpoint ?? '' },
    );
  }

  public markNotificationsAsRead(ids?: number[]): Observable<APIResponse<{ success: boolean }>> {
    return this.http.post<APIResponse<{ success: boolean }>>(`${this.baseUrl}/notifications/read`, {
      ids: ids ?? [],
    });
  }
}
