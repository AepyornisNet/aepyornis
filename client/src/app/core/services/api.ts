import { inject, Injectable } from '@angular/core';
import { HttpClient, HttpParams, HttpResponse } from '@angular/common/http';
import { Observable } from 'rxjs';
import {
  APIResponse,
  PaginatedAPIResponse,
  PaginationParams,
} from '../../core/types/api-response';
import {
  AppConfig,
  AppInfo,
  FollowRequest,
  FullUserProfile,
  ProfileUpdateRequest,
  UserProfile,
  UserUpdateRequest,
} from '../../core/types/user';
import {
  CalendarEvent,
  ClimbRecordEntry,
  DistanceRecordEntry,
  Totals,
  Workout,
  WorkoutBreakdown,
  WorkoutDetail,
  WorkoutListParams,
  WorkoutRangeStats,
  WorkoutRecord,
} from '../../core/types/workout';
import { Measurement } from '../../core/types/measurement';
import { Equipment } from '../../core/types/equipment';
import { RouteSegment, RouteSegmentDetail } from '../../core/types/route-segment';
import {
  GeoJsonFeatureCollection,
  HeatmapCoordinateList,
  Statistics,
  StatisticsParams,
} from '../../core/types/statistics';
import { RegisterRequest, SignInRequest } from '../../core/types/auth';

@Injectable({
  providedIn: 'root',
})
export class Api {
  private http = inject(HttpClient);

  private baseUrl = '/api/v2';

  // Auth endpoints
  public signIn(payload: SignInRequest): Observable<APIResponse<UserProfile>> {
    return this.http.post<APIResponse<UserProfile>>(`${this.baseUrl}/auth/signin`, payload);
  }

  public register(payload: RegisterRequest): Observable<APIResponse<{ message: string }>> {
    return this.http.post<APIResponse<{ message: string }>>(`${this.baseUrl}/auth/register`, payload);
  }

  public signOut(): Observable<APIResponse<{ message: string }>> {
    return this.http.post<APIResponse<{ message: string }>>(`${this.baseUrl}/auth/signout`, {});
  }

  public whoami(): Observable<APIResponse<UserProfile>> {
    return this.http.get<APIResponse<UserProfile>>(`${this.baseUrl}/whoami`);
  }

  public getAppInfo(): Observable<APIResponse<AppInfo>> {
    return this.http.get<APIResponse<AppInfo>>(`${this.baseUrl}/app-info`);
  }

  // Workouts endpoints
  public getWorkouts(params?: WorkoutListParams): Observable<PaginatedAPIResponse<Workout>> {
    let httpParams = new HttpParams();
    if (params?.page) {
      httpParams = httpParams.set('page', params.page.toString());
    }
    if (params?.per_page) {
      httpParams = httpParams.set('per_page', params.per_page.toString());
    }
    if (params?.type) {
      httpParams = httpParams.set('type', params.type);
    }
    if (params?.active !== undefined) {
      httpParams = httpParams.set('active', params.active ? 'true' : 'false');
    }
    if (params?.since) {
      httpParams = httpParams.set('since', params.since);
    }
    if (params?.order_by) {
      httpParams = httpParams.set('order_by', params.order_by);
    }
    if (params?.order_dir) {
      httpParams = httpParams.set('order_dir', params.order_dir);
    }
    return this.http.get<PaginatedAPIResponse<Workout>>(`${this.baseUrl}/workouts`, {
      params: httpParams,
    });
  }

  public getWorkout(id: number): Observable<APIResponse<WorkoutDetail>> {
    return this.http.get<APIResponse<WorkoutDetail>>(`${this.baseUrl}/workouts/${id}`);
  }

  public getWorkoutBreakdown(
    id: number,
    params?: { count?: number; mode?: 'auto' | 'laps' | 'unit' | string },
  ): Observable<APIResponse<WorkoutBreakdown>> {
    let httpParams = new HttpParams();

    if (params?.count) {
      httpParams = httpParams.set('count', params.count.toString());
    }

    if (params?.mode) {
      httpParams = httpParams.set('mode', params.mode);
    }

    return this.http.get<APIResponse<WorkoutBreakdown>>(
      `${this.baseUrl}/workouts/${id}/breakdown`,
      { params: httpParams },
    );
  }

  public getWorkoutRangeStats(
    id: number,
    params?: { start_index?: number; end_index?: number },
  ): Observable<APIResponse<WorkoutRangeStats>> {
    let httpParams = new HttpParams();

    if (params?.start_index !== undefined) {
      httpParams = httpParams.set('start_index', params.start_index.toString());
    }

    if (params?.end_index !== undefined) {
      httpParams = httpParams.set('end_index', params.end_index.toString());
    }

    return this.http.get<APIResponse<WorkoutRangeStats>>(
      `${this.baseUrl}/workouts/${id}/stats-range`,
      { params: httpParams },
    );
  }

  public getRecentWorkouts(limit?: number, offset?: number): Observable<APIResponse<Workout[]>> {
    let httpParams = new HttpParams();
    if (limit) {
      httpParams = httpParams.set('limit', limit.toString());
    }
    if (offset !== undefined) {
      httpParams = httpParams.set('offset', offset.toString());
    }
    return this.http.get<APIResponse<Workout[]>>(`${this.baseUrl}/workouts/recent`, {
      params: httpParams,
    });
  }

  public createWorkoutFromFile(formData: FormData): Observable<APIResponse<Workout[]>> {
    return this.http.post<APIResponse<Workout[]>>(`${this.baseUrl}/workouts`, formData);
  }

  public createWorkoutManual(workout: {
    name: string;
    date: string;
    timezone: string;
    location?: string;
    duration_hours?: number;
    duration_minutes?: number;
    duration_seconds?: number;
    distance?: number;
    repetitions?: number;
    weight?: number;
    notes?: string;
    type: string;
    visibility?: '' | 'followers' | 'public';
    custom_type?: string;
    equipment_ids?: number[];
  }): Observable<APIResponse<Workout>> {
    return this.http.post<APIResponse<Workout>>(`${this.baseUrl}/workouts`, workout);
  }

  public updateWorkout(
    id: number,
    workout: {
      name?: string;
      date?: string;
      timezone?: string;
      location?: string;
      duration_hours?: number;
      duration_minutes?: number;
      duration_seconds?: number;
      distance?: number;
      repetitions?: number;
      weight?: number;
      notes?: string;
      type?: string;
      visibility?: '' | 'followers' | 'public';
      custom_type?: string;
      equipment_ids?: number[];
    },
  ): Observable<APIResponse<Workout>> {
    return this.http.put<APIResponse<Workout>>(`${this.baseUrl}/workouts/${id}`, workout);
  }

  public deleteWorkout(id: number): Observable<APIResponse<{ message: string }>> {
    return this.http.delete<APIResponse<{ message: string }>>(`${this.baseUrl}/workouts/${id}`);
  }

  public toggleWorkoutLock(id: number): Observable<APIResponse<Workout>> {
    return this.http.post<APIResponse<Workout>>(`${this.baseUrl}/workouts/${id}/toggle-lock`, {});
  }

  public refreshWorkout(id: number): Observable<APIResponse<{ message: string }>> {
    return this.http.post<APIResponse<{ message: string }>>(
      `${this.baseUrl}/workouts/${id}/refresh`,
      {},
    );
  }

  public downloadWorkout(id: number): Observable<HttpResponse<Blob>> {
    return this.http.get(`${this.baseUrl}/workouts/${id}/download`, {
      observe: 'response',
      responseType: 'blob',
    });
  }

  // Measurements endpoints
  public getMeasurements(params?: PaginationParams): Observable<PaginatedAPIResponse<Measurement>> {
    let httpParams = new HttpParams();
    if (params?.page) {
      httpParams = httpParams.set('page', params.page.toString());
    }
    if (params?.per_page) {
      httpParams = httpParams.set('per_page', params.per_page.toString());
    }
    return this.http.get<PaginatedAPIResponse<Measurement>>(`${this.baseUrl}/measurements`, {
      params: httpParams,
    });
  }

  public createOrUpdateMeasurement(measurement: {
    date: string;
    weight?: number;
    height?: number;
    steps?: number;
    ftp?: number;
    resting_heart_rate?: number;
    max_heart_rate?: number;
  }): Observable<APIResponse<Measurement>> {
    return this.http.post<APIResponse<Measurement>>(`${this.baseUrl}/measurements`, measurement);
  }

  public deleteMeasurement(date: string): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/measurements/${date}`);
  }

  // Equipment endpoints
  public getEquipment(params?: PaginationParams): Observable<PaginatedAPIResponse<Equipment>> {
    let httpParams = new HttpParams();
    if (params?.page) {
      httpParams = httpParams.set('page', params.page.toString());
    }
    if (params?.per_page) {
      httpParams = httpParams.set('per_page', params.per_page.toString());
    }
    return this.http.get<PaginatedAPIResponse<Equipment>>(`${this.baseUrl}/equipment`, {
      params: httpParams,
    });
  }

  public getEquipmentById(id: number): Observable<APIResponse<Equipment>> {
    return this.http.get<APIResponse<Equipment>>(`${this.baseUrl}/equipment/${id}`);
  }

  public createEquipment(equipment: Partial<Equipment>): Observable<APIResponse<Equipment>> {
    return this.http.post<APIResponse<Equipment>>(`${this.baseUrl}/equipment`, equipment);
  }

  public updateEquipment(
    id: number,
    equipment: Partial<Equipment>,
  ): Observable<APIResponse<Equipment>> {
    return this.http.put<APIResponse<Equipment>>(`${this.baseUrl}/equipment/${id}`, equipment);
  }

  public deleteEquipment(id: number): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/equipment/${id}`);
  }

  // Route segments endpoints
  public getRouteSegments(
    params?: PaginationParams,
  ): Observable<PaginatedAPIResponse<RouteSegment>> {
    let httpParams = new HttpParams();
    if (params?.page) {
      httpParams = httpParams.set('page', params.page.toString());
    }
    if (params?.per_page) {
      httpParams = httpParams.set('per_page', params.per_page.toString());
    }
    return this.http.get<PaginatedAPIResponse<RouteSegment>>(`${this.baseUrl}/route-segments`, {
      params: httpParams,
    });
  }

  public getRouteSegment(id: number): Observable<APIResponse<RouteSegmentDetail>> {
    return this.http.get<APIResponse<RouteSegmentDetail>>(`${this.baseUrl}/route-segments/${id}`);
  }

  public createRouteSegment(formData: FormData): Observable<APIResponse<RouteSegment[]>> {
    return this.http.post<APIResponse<RouteSegment[]>>(`${this.baseUrl}/route-segments`, formData);
  }

  public createRouteSegmentFromWorkout(
    workoutId: number,
    params: {
      name: string;
      start: number;
      end: number;
    },
  ): Observable<APIResponse<RouteSegmentDetail>> {
    return this.http.post<APIResponse<RouteSegmentDetail>>(
      `${this.baseUrl}/workouts/${workoutId}/route-segment`,
      params,
    );
  }

  public updateRouteSegment(
    id: number,
    params: {
      name: string;
      notes: string;
      bidirectional: boolean;
      circular: boolean;
    },
  ): Observable<APIResponse<RouteSegmentDetail>> {
    return this.http.put<APIResponse<RouteSegmentDetail>>(
      `${this.baseUrl}/route-segments/${id}`,
      params,
    );
  }

  public deleteRouteSegment(id: number): Observable<APIResponse<{ message: string }>> {
    return this.http.delete<APIResponse<{ message: string }>>(
      `${this.baseUrl}/route-segments/${id}`,
    );
  }

  public refreshRouteSegment(id: number): Observable<APIResponse<{ message: string }>> {
    return this.http.post<APIResponse<{ message: string }>>(
      `${this.baseUrl}/route-segments/${id}/refresh`,
      {},
    );
  }

  public findRouteSegmentMatches(id: number): Observable<APIResponse<{ message: string }>> {
    return this.http.post<APIResponse<{ message: string }>>(
      `${this.baseUrl}/route-segments/${id}/matches`,
      {},
    );
  }

  public downloadRouteSegment(id: number): Observable<Blob> {
    return this.http.get(`${this.baseUrl}/route-segments/${id}/download`, {
      responseType: 'blob',
    });
  }

  // Dashboard endpoints
  public getTotals(): Observable<APIResponse<Totals>> {
    return this.http.get<APIResponse<Totals>>(`${this.baseUrl}/totals`);
  }

  public getRecords(): Observable<APIResponse<WorkoutRecord[]>> {
    return this.http.get<APIResponse<WorkoutRecord[]>>(`${this.baseUrl}/records`);
  }

  public getDistanceRecordRanking(params: {
    workout_type: string;
    label: string;
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

    return this.http.get<PaginatedAPIResponse<DistanceRecordEntry>>(
      `${this.baseUrl}/records/ranking`,
      { params: httpParams },
    );
  }

  public getClimbRanking(params: {
    workout_type: string;
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

  // Heatmap endpoints
  public getWorkoutsCoordinates(): Observable<APIResponse<HeatmapCoordinateList>> {
    return this.http.get<APIResponse<HeatmapCoordinateList>>(
      `${this.baseUrl}/workouts/coordinates`,
    );
  }

  public getWorkoutsCenters(): Observable<APIResponse<GeoJsonFeatureCollection>> {
    return this.http.get<APIResponse<GeoJsonFeatureCollection>>(`${this.baseUrl}/workouts/centers`);
  }

  // Calendar endpoints
  public getCalendarEvents(params?: {
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
    return this.http.get<APIResponse<CalendarEvent[]>>(`${this.baseUrl}/workouts/calendar`, {
      params: httpParams,
    });
  }
}
