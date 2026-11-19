import { inject, Injectable } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';
import { APIResponse, PaginatedAPIResponse } from '../../../core/types/api-response';
import {
  ActivityPubActor,
  ActivityPubProfileSummary,
  AppConfig,
  AppInfo,
  FollowRequest,
  FullUserProfile,
  HammerheadConnectionStatus,
  ProfileChangePasswordRequest,
  ProfileUpdateRequest,
  UserProfile,
  UserUpdateRequest,
} from '../../../core/types/user';
import {
  CalendarEvent,
  ClimbRecordEntry,
  DistanceRecordEntry,
  Totals,
  WorkoutRecord,
} from '../../../core/types/workout';
import {
  GeoJsonFeatureCollection,
  HeatmapCoordinateList,
  Statistics,
  StatisticsParams,
} from '../../../core/types/statistics';

@Injectable({
  providedIn: 'root',
})
export class ProfileApiService {
  private http = inject(HttpClient);
  private baseUrl = '/api/v2';

  // Dashboard endpoints
  public getTotals(handle?: string): Observable<APIResponse<Totals[]>> {
    let httpParams = new HttpParams();
    if (handle) {
      httpParams = httpParams.set('handle', handle);
    }
    return this.http.get<APIResponse<Totals[]>>(`${this.baseUrl}/totals`, {
      params: httpParams,
    });
  }

  public getRecords(handle?: string): Observable<APIResponse<WorkoutRecord[]>> {
    let httpParams = new HttpParams();
    if (handle) {
      httpParams = httpParams.set('handle', handle);
    }
    return this.http.get<APIResponse<WorkoutRecord[]>>(`${this.baseUrl}/records`, {
      params: httpParams,
    });
  }

  public getDistanceRecordRanking(params: {
    workout_type: string;
    label: string;
    handle?: string;
    start?: string;
    end?: string;
    page?: number;
    per_page?: number;
  }): Observable<PaginatedAPIResponse<DistanceRecordEntry>> {
    let httpParams = new HttpParams()
      .set('workout_type', params.workout_type)
      .set('label', params.label);

    if (params.start) {
      httpParams = httpParams.set('start', params.start);
    }
    if (params.end) {
      httpParams = httpParams.set('end', params.end);
    }
    if (params.page) {
      httpParams = httpParams.set('page', params.page.toString());
    }
    if (params.per_page) {
      httpParams = httpParams.set('per_page', params.per_page.toString());
    }
    if (params.handle) {
      httpParams = httpParams.set('handle', params.handle);
    }

    return this.http.get<PaginatedAPIResponse<DistanceRecordEntry>>(
      `${this.baseUrl}/records/ranking`,
      { params: httpParams },
    );
  }

  public getClimbRanking(params: {
    workout_type: string;
    handle?: string;
    start?: string;
    end?: string;
    page?: number;
    per_page?: number;
  }): Observable<PaginatedAPIResponse<ClimbRecordEntry>> {
    let httpParams = new HttpParams().set('workout_type', params.workout_type);

    if (params.start) {
      httpParams = httpParams.set('start', params.start);
    }
    if (params.end) {
      httpParams = httpParams.set('end', params.end);
    }
    if (params.page) {
      httpParams = httpParams.set('page', params.page.toString());
    }
    if (params.per_page) {
      httpParams = httpParams.set('per_page', params.per_page.toString());
    }
    if (params.handle) {
      httpParams = httpParams.set('handle', params.handle);
    }

    return this.http.get<PaginatedAPIResponse<ClimbRecordEntry>>(
      `${this.baseUrl}/records/climbs/ranking`,
      { params: httpParams },
    );
  }

  // Profile endpoints
  public getProfile(): Observable<APIResponse<FullUserProfile>> {
    return this.http.get<APIResponse<FullUserProfile>>(`${this.baseUrl}/profile`);
  }

  public updateProfile(profile: ProfileUpdateRequest): Observable<APIResponse<FullUserProfile>> {
    return this.http.put<APIResponse<FullUserProfile>>(`${this.baseUrl}/profile`, profile);
  }

  public changePassword(
    payload: ProfileChangePasswordRequest,
  ): Observable<APIResponse<{ message: string }>> {
    return this.http.post<APIResponse<{ message: string }>>(
      `${this.baseUrl}/profile/change-password`,
      payload,
    );
  }

  public resetAPIKey(): Observable<APIResponse<{ api_key: string; message: string }>> {
    return this.http.post<APIResponse<{ api_key: string; message: string }>>(
      `${this.baseUrl}/profile/reset-api-key`,
      {},
    );
  }

  public enableActivityPub(): Observable<APIResponse<{ activity_pub: boolean; message: string }>> {
    return this.http.post<APIResponse<{ activity_pub: boolean; message: string }>>(
      `${this.baseUrl}/profile/enable-activity-pub`,
      {},
    );
  }

  public getFollowRequests(): Observable<APIResponse<FollowRequest[]>> {
    return this.http.get<APIResponse<FollowRequest[]>>(`${this.baseUrl}/profile/follow-requests`);
  }

  public getUserProfileSummary(
    handle?: string,
  ): Observable<APIResponse<ActivityPubProfileSummary>> {
    let httpParams = new HttpParams();
    if (handle) {
      httpParams = httpParams.set('handle', handle);
    }

    return this.http.get<APIResponse<ActivityPubProfileSummary>>(`${this.baseUrl}/user-profile`, {
      params: httpParams,
    });
  }

  public searchProfiles(query: string): Observable<APIResponse<ActivityPubProfileSummary[]>> {
    const params = new HttpParams().set('q', query);
    return this.http.get<APIResponse<ActivityPubProfileSummary[]>>(
      `${this.baseUrl}/user-profile/search`,
      { params },
    );
  }

  public getLocalActivityPubActor(username: string): Observable<ActivityPubActor> {
    return this.http.get<ActivityPubActor>(`/ap/users/${encodeURIComponent(username)}`);
  }

  public followUserByHandle(handle: string): Observable<APIResponse<ActivityPubProfileSummary>> {
    const params = new HttpParams().set('handle', handle);
    return this.http.post<APIResponse<ActivityPubProfileSummary>>(
      `${this.baseUrl}/user-profile/follow`,
      {},
      { params },
    );
  }

  public unfollowUserByHandle(handle: string): Observable<APIResponse<ActivityPubProfileSummary>> {
    const params = new HttpParams().set('handle', handle);
    return this.http.post<APIResponse<ActivityPubProfileSummary>>(
      `${this.baseUrl}/user-profile/unfollow`,
      {},
      { params },
    );
  }

  public acceptFollowRequest(id: number): Observable<APIResponse<FollowRequest>> {
    return this.http.post<APIResponse<FollowRequest>>(
      `${this.baseUrl}/profile/follow-requests/${id}/accept`,
      {},
    );
  }

  public refreshWorkouts(): Observable<APIResponse<{ message: string }>> {
    return this.http.post<APIResponse<{ message: string }>>(
      `${this.baseUrl}/profile/refresh-workouts`,
      {},
    );
  }

  public getHammerheadConnection(): Observable<APIResponse<HammerheadConnectionStatus>> {
    return this.http.get<APIResponse<HammerheadConnectionStatus>>(
      `${this.baseUrl}/profile/apps/hammerhead`,
    );
  }

  public connectHammerhead(): Observable<APIResponse<{ authorize_url: string }>> {
    return this.http.post<APIResponse<{ authorize_url: string }>>(
      `${this.baseUrl}/profile/apps/hammerhead/connect`,
      {},
    );
  }

  public disconnectHammerhead(): Observable<APIResponse<{ message: string }>> {
    return this.http.delete<APIResponse<{ message: string }>>(
      `${this.baseUrl}/profile/apps/hammerhead`,
    );
  }

  // Admin endpoints
  public getUsers(): Observable<APIResponse<UserProfile[]>> {
    return this.http.get<APIResponse<UserProfile[]>>(`${this.baseUrl}/admin/users`);
  }

  public getUser(id: number): Observable<APIResponse<UserProfile>> {
    return this.http.get<APIResponse<UserProfile>>(`${this.baseUrl}/admin/users/${id}`);
  }

  public updateUser(id: number, user: UserUpdateRequest): Observable<APIResponse<UserProfile>> {
    return this.http.put<APIResponse<UserProfile>>(`${this.baseUrl}/admin/users/${id}`, user);
  }

  public deleteUser(id: number): Observable<APIResponse<{ message: string }>> {
    return this.http.delete<APIResponse<{ message: string }>>(`${this.baseUrl}/admin/users/${id}`);
  }

  public updateAppConfig(config: AppConfig): Observable<APIResponse<AppInfo>> {
    return this.http.put<APIResponse<AppInfo>>(`${this.baseUrl}/admin/config`, config);
  }

  public getAppInfo(): Observable<APIResponse<AppInfo>> {
    return this.http.get<APIResponse<AppInfo>>(`${this.baseUrl}/app-info`);
  }

  // Statistics endpoints
  public getStatistics(params?: StatisticsParams): Observable<APIResponse<Statistics>> {
    let httpParams = new HttpParams();
    if (params?.since) {
      httpParams = httpParams.set('since', params.since);
    }
    if (params?.per) {
      httpParams = httpParams.set('per', params.per);
    }
    return this.http.get<APIResponse<Statistics>>(`${this.baseUrl}/statistics`, {
      params: httpParams,
    });
  }

  public getWorkoutsCoordinates(params?: {
    cellSize?: number;
    minLat?: number;
    minLng?: number;
    maxLat?: number;
    maxLng?: number;
    type?: string;
    sub_type?: string;
    since?: string;
  }): Observable<APIResponse<HeatmapCoordinateList>> {
    let httpParams = new HttpParams();
    if (params?.cellSize !== undefined) {
      httpParams = httpParams.set('cell_size', params.cellSize.toString());
    }
    if (params?.minLat !== undefined) {
      httpParams = httpParams.set('min_lat', params.minLat.toString());
    }
    if (params?.minLng !== undefined) {
      httpParams = httpParams.set('min_lng', params.minLng.toString());
    }
    if (params?.maxLat !== undefined) {
      httpParams = httpParams.set('max_lat', params.maxLat.toString());
    }
    if (params?.maxLng !== undefined) {
      httpParams = httpParams.set('max_lng', params.maxLng.toString());
    }
    if (params?.type) {
      httpParams = httpParams.set('type', params.type);
    }
    if (params?.sub_type) {
      httpParams = httpParams.set('sub_type', params.sub_type);
    }
    if (params?.since) {
      httpParams = httpParams.set('since', params.since);
    }
    return this.http.get<APIResponse<HeatmapCoordinateList>>(
      `${this.baseUrl}/workouts/coordinates`,
      { params: httpParams },
    );
  }

  public getWorkoutsCenters(params?: {
    type?: string;
    sub_type?: string;
    since?: string;
  }): Observable<APIResponse<GeoJsonFeatureCollection>> {
    let httpParams = new HttpParams();
    if (params?.type) {
      httpParams = httpParams.set('type', params.type);
    }
    if (params?.sub_type) {
      httpParams = httpParams.set('sub_type', params.sub_type);
    }
    if (params?.since) {
      httpParams = httpParams.set('since', params.since);
    }
    return this.http.get<APIResponse<GeoJsonFeatureCollection>>(
      `${this.baseUrl}/workouts/centers`,
      { params: httpParams },
    );
  }

  // Calendar endpoints
  public getCalendarEvents(params?: {
    handle?: string;
    start?: string;
    end?: string;
    timeZone?: string;
  }): Observable<APIResponse<CalendarEvent[]>> {
    let httpParams = new HttpParams();
    if (params?.start) {
      httpParams = httpParams.set('start', params.start);
    }
    if (params?.end) {
      httpParams = httpParams.set('end', params.end);
    }
    if (params?.timeZone) {
      httpParams = httpParams.set('timeZone', params.timeZone);
    }
    if (params?.handle) {
      httpParams = httpParams.set('handle', params.handle);
    }
    return this.http.get<APIResponse<CalendarEvent[]>>(`${this.baseUrl}/workouts/calendar`, {
      params: httpParams,
    });
  }

  // Legal endpoints
  public getLegalNotice(lang: string): Observable<string> {
    return this.http.get(`${this.baseUrl}/legal/notice/${lang}`, { responseType: 'text' });
  }

  public getPrivacyPolicy(lang: string): Observable<string> {
    return this.http.get(`${this.baseUrl}/legal/privacy/${lang}`, { responseType: 'text' });
  }
}
