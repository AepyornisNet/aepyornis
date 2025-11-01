import { inject, Injectable } from '@angular/core';
import { HttpClient, HttpParams, HttpResponse } from '@angular/common/http';
import { Observable } from 'rxjs';
import { APIResponse, PaginatedAPIResponse, PaginationParams } from '../../core/types/api-response';
import {
  AppConfig,
  AppInfo,
  FullUserProfile,
  ProfileUpdateRequest,
  UserProfile,
  UserUpdateRequest,
} from '../../core/types/user';
import {
  CalendarEvent,
  Totals,
  Workout,
  WorkoutDetail,
  WorkoutRecord,
} from '../../core/types/workout';
import { Measurement } from '../../core/types/measurement';
import { Equipment } from '../../core/types/equipment';
import { RouteSegment, RouteSegmentDetail } from '../../core/types/route-segment';
import {
  GeoJsonFeatureCollection,
  Statistics,
  StatisticsParams,
} from '../../core/types/statistics';

@Injectable({
  providedIn: 'root',
})
export class Api {
  private http = inject(HttpClient);

  private baseUrl = '/api/v2';

  whoami(): Observable<APIResponse<UserProfile>> {
    return this.http.get<APIResponse<UserProfile>>(`${this.baseUrl}/whoami`);
  }

  getAppInfo(): Observable<APIResponse<AppInfo>> {
    return this.http.get<APIResponse<AppInfo>>(`${this.baseUrl}/app-info`);
  }

  // Workouts endpoints
  getWorkouts(params?: PaginationParams): Observable<PaginatedAPIResponse<Workout>> {
    let httpParams = new HttpParams();
    if (params?.page) {
      httpParams = httpParams.set('page', params.page.toString());
    }
    if (params?.per_page) {
      httpParams = httpParams.set('per_page', params.per_page.toString());
    }
    return this.http.get<PaginatedAPIResponse<Workout>>(`${this.baseUrl}/workouts`, {
      params: httpParams,
    });
  }

  getWorkout(id: number): Observable<APIResponse<WorkoutDetail>> {
    return this.http.get<APIResponse<WorkoutDetail>>(`${this.baseUrl}/workouts/${id}`);
  }

  getPublicWorkout(uuid: string): Observable<APIResponse<WorkoutDetail>> {
    return this.http.get<APIResponse<WorkoutDetail>>(`${this.baseUrl}/workouts/public/${uuid}`);
  }

  getRecentWorkouts(limit?: number, offset?: number): Observable<APIResponse<Workout[]>> {
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

  createWorkoutFromFile(formData: FormData): Observable<APIResponse<Workout[]>> {
    return this.http.post<APIResponse<Workout[]>>(`${this.baseUrl}/workouts`, formData);
  }

  createWorkoutManual(workout: {
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
    custom_type?: string;
    equipment_ids?: number[];
  }): Observable<APIResponse<Workout>> {
    return this.http.post<APIResponse<Workout>>(`${this.baseUrl}/workouts`, workout);
  }

  updateWorkout(
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
      custom_type?: string;
      equipment_ids?: number[];
    },
  ): Observable<APIResponse<Workout>> {
    return this.http.put<APIResponse<Workout>>(`${this.baseUrl}/workouts/${id}`, workout);
  }

  deleteWorkout(id: number): Observable<APIResponse<{ message: string }>> {
    return this.http.delete<APIResponse<{ message: string }>>(`${this.baseUrl}/workouts/${id}`);
  }

  toggleWorkoutLock(id: number): Observable<APIResponse<Workout>> {
    return this.http.post<APIResponse<Workout>>(`${this.baseUrl}/workouts/${id}/toggle-lock`, {});
  }

  refreshWorkout(id: number): Observable<APIResponse<{ message: string }>> {
    return this.http.post<APIResponse<{ message: string }>>(
      `${this.baseUrl}/workouts/${id}/refresh`,
      {},
    );
  }

  shareWorkout(
    id: number,
  ): Observable<APIResponse<{ message: string; public_uuid: string; share_url: string }>> {
    return this.http.post<APIResponse<{ message: string; public_uuid: string; share_url: string }>>(
      `${this.baseUrl}/workouts/${id}/share`,
      {},
    );
  }

  deleteWorkoutShare(id: number): Observable<APIResponse<{ message: string }>> {
    return this.http.delete<APIResponse<{ message: string }>>(
      `${this.baseUrl}/workouts/${id}/share`,
    );
  }

  downloadWorkout(id: number): Observable<HttpResponse<Blob>> {
    return this.http.get(`${this.baseUrl}/workouts/${id}/download`, {
      observe: 'response',
      responseType: 'blob',
    });
  }

  // Measurements endpoints
  getMeasurements(params?: PaginationParams): Observable<PaginatedAPIResponse<Measurement>> {
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

  createOrUpdateMeasurement(measurement: {
    date: string;
    weight?: number;
    height?: number;
    steps?: number;
  }): Observable<APIResponse<Measurement>> {
    return this.http.post<APIResponse<Measurement>>(`${this.baseUrl}/measurements`, measurement);
  }

  deleteMeasurement(date: string): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/measurements/${date}`);
  }

  // Equipment endpoints
  getEquipment(params?: PaginationParams): Observable<PaginatedAPIResponse<Equipment>> {
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

  getEquipmentById(id: number): Observable<APIResponse<Equipment>> {
    return this.http.get<APIResponse<Equipment>>(`${this.baseUrl}/equipment/${id}`);
  }

  createEquipment(equipment: Partial<Equipment>): Observable<APIResponse<Equipment>> {
    return this.http.post<APIResponse<Equipment>>(`${this.baseUrl}/equipment`, equipment);
  }

  updateEquipment(id: number, equipment: Partial<Equipment>): Observable<APIResponse<Equipment>> {
    return this.http.put<APIResponse<Equipment>>(`${this.baseUrl}/equipment/${id}`, equipment);
  }

  deleteEquipment(id: number): Observable<void> {
    return this.http.delete<void>(`${this.baseUrl}/equipment/${id}`);
  }

  // Route segments endpoints
  getRouteSegments(params?: PaginationParams): Observable<PaginatedAPIResponse<RouteSegment>> {
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

  getRouteSegment(id: number): Observable<APIResponse<RouteSegmentDetail>> {
    return this.http.get<APIResponse<RouteSegmentDetail>>(`${this.baseUrl}/route-segments/${id}`);
  }

  createRouteSegmentFromWorkout(
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

  updateRouteSegment(
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

  deleteRouteSegment(id: number): Observable<APIResponse<{ message: string }>> {
    return this.http.delete<APIResponse<{ message: string }>>(
      `${this.baseUrl}/route-segments/${id}`,
    );
  }

  refreshRouteSegment(id: number): Observable<APIResponse<{ message: string }>> {
    return this.http.post<APIResponse<{ message: string }>>(
      `${this.baseUrl}/route-segments/${id}/refresh`,
      {},
    );
  }

  findRouteSegmentMatches(id: number): Observable<APIResponse<{ message: string }>> {
    return this.http.post<APIResponse<{ message: string }>>(
      `${this.baseUrl}/route-segments/${id}/matches`,
      {},
    );
  }

  downloadRouteSegment(id: number): Observable<Blob> {
    return this.http.get(`${this.baseUrl}/route-segments/${id}/download`, {
      responseType: 'blob',
    });
  }

  // Dashboard endpoints
  getTotals(): Observable<APIResponse<Totals>> {
    return this.http.get<APIResponse<Totals>>(`${this.baseUrl}/totals`);
  }

  getRecords(): Observable<APIResponse<WorkoutRecord[]>> {
    return this.http.get<APIResponse<WorkoutRecord[]>>(`${this.baseUrl}/records`);
  }

  // Profile endpoints
  getProfile(): Observable<APIResponse<FullUserProfile>> {
    return this.http.get<APIResponse<FullUserProfile>>(`${this.baseUrl}/profile`);
  }

  updateProfile(profile: ProfileUpdateRequest): Observable<APIResponse<FullUserProfile>> {
    return this.http.put<APIResponse<FullUserProfile>>(`${this.baseUrl}/profile`, profile);
  }

  resetAPIKey(): Observable<APIResponse<{ api_key: string; message: string }>> {
    return this.http.post<APIResponse<{ api_key: string; message: string }>>(
      `${this.baseUrl}/profile/reset-api-key`,
      {},
    );
  }

  refreshWorkouts(): Observable<APIResponse<{ message: string }>> {
    return this.http.post<APIResponse<{ message: string }>>(
      `${this.baseUrl}/profile/refresh-workouts`,
      {},
    );
  }

  // Admin endpoints
  getUsers(): Observable<APIResponse<UserProfile[]>> {
    return this.http.get<APIResponse<UserProfile[]>>(`${this.baseUrl}/admin/users`);
  }

  getUser(id: number): Observable<APIResponse<UserProfile>> {
    return this.http.get<APIResponse<UserProfile>>(`${this.baseUrl}/admin/users/${id}`);
  }

  updateUser(id: number, user: UserUpdateRequest): Observable<APIResponse<UserProfile>> {
    return this.http.put<APIResponse<UserProfile>>(`${this.baseUrl}/admin/users/${id}`, user);
  }

  deleteUser(id: number): Observable<APIResponse<{ message: string }>> {
    return this.http.delete<APIResponse<{ message: string }>>(`${this.baseUrl}/admin/users/${id}`);
  }

  updateAppConfig(config: AppConfig): Observable<APIResponse<AppInfo>> {
    return this.http.put<APIResponse<AppInfo>>(`${this.baseUrl}/admin/config`, config);
  }

  // Statistics endpoints
  getStatistics(params?: StatisticsParams): Observable<APIResponse<Statistics>> {
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
  getWorkoutsCoordinates(): Observable<APIResponse<GeoJsonFeatureCollection>> {
    return this.http.get<APIResponse<GeoJsonFeatureCollection>>(
      `${this.baseUrl}/workouts/coordinates`,
    );
  }

  getWorkoutsCenters(): Observable<APIResponse<GeoJsonFeatureCollection>> {
    return this.http.get<APIResponse<GeoJsonFeatureCollection>>(`${this.baseUrl}/workouts/centers`);
  }

  // Calendar endpoints
  getCalendarEvents(params?: {
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
