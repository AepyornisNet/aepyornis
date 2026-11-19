import { inject, Injectable } from '@angular/core';
import { HttpClient, HttpParams, HttpResponse } from '@angular/common/http';
import { Observable } from 'rxjs';
import {
  APIResponse,
  PaginatedAPIResponse,
  PaginationParams,
} from '../../../core/types/api-response';
import {
  Workout,
  WorkoutBreakdown,
  WorkoutDetail,
  WorkoutListParams,
  WorkoutRangeStats,
  WorkoutReply,
} from '../../../core/types/workout';
import { Like } from '../../../core/types/like';

@Injectable({
  providedIn: 'root',
})
export class WorkoutApiService {
  private http = inject(HttpClient);
  private baseUrl = '/api/v2';

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
    if (params?.sub_type) {
      httpParams = httpParams.set('sub_type', params.sub_type);
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

  public getRecentWorkouts(
    limit?: number,
    offset?: number,
    scope?: 'following' | 'global',
    handle?: string,
  ): Observable<APIResponse<Workout[]>> {
    let httpParams = new HttpParams();
    if (limit) {
      httpParams = httpParams.set('limit', limit.toString());
    }
    if (offset !== undefined) {
      httpParams = httpParams.set('offset', offset.toString());
    }
    if (scope) {
      httpParams = httpParams.set('scope', scope);
    }
    if (handle) {
      httpParams = httpParams.set('handle', handle);
    }
    return this.http.get<APIResponse<Workout[]>>(`${this.baseUrl}/workouts/recent`, {
      params: httpParams,
    });
  }

  public getWorkoutFilterOptions(): Observable<
    APIResponse<{ types: string[]; sub_types_by_type: Record<string, string[]> }>
  > {
    return this.http.get<
      APIResponse<{ types: string[]; sub_types_by_type: Record<string, string[]> }>
    >(`${this.baseUrl}/workouts/filter-options`);
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

  public downloadWorkoutsZip(ids: number[]): Observable<Blob> {
    const params = ids.map((id) => `ids=${id}`).join('&');
    return this.http.get(`${this.baseUrl}/workouts/download-zip?${params}`, {
      responseType: 'blob',
    });
  }

  public addEquipmentToWorkouts(
    workoutIds: number[],
    equipmentIds: number[],
  ): Observable<APIResponse<{ message: string }>> {
    return this.http.post<APIResponse<{ message: string }>>(
      `${this.baseUrl}/workouts/add-equipment`,
      {
        workout_ids: workoutIds,
        equipment_ids: equipmentIds,
      },
    );
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

  public likeWorkout(
    id: number,
  ): Observable<APIResponse<{ workout_id: number; likes_count: number; liked: boolean }>> {
    return this.http.post<APIResponse<{ workout_id: number; likes_count: number; liked: boolean }>>(
      `${this.baseUrl}/workouts/${id}/like`,
      {},
    );
  }

  public getWorkoutLikes(workoutId: number): Observable<APIResponse<Like[]>> {
    return this.http.get<APIResponse<Like[]>>(`${this.baseUrl}/workouts/${workoutId}/likes`);
  }

  public createReply(workoutId: number, content: string): Observable<APIResponse<WorkoutReply>> {
    return this.http.post<APIResponse<WorkoutReply>>(
      `${this.baseUrl}/workouts/${workoutId}/replies`,
      { content },
    );
  }

  public getWorkoutReplies(
    workoutId: number,
    params?: PaginationParams,
  ): Observable<PaginatedAPIResponse<WorkoutReply>> {
    let httpParams = new HttpParams();
    if (params?.page) {
      httpParams = httpParams.set('page', params.page.toString());
    }
    if (params?.per_page) {
      httpParams = httpParams.set('per_page', params.per_page.toString());
    }

    return this.http.get<PaginatedAPIResponse<WorkoutReply>>(
      `${this.baseUrl}/workouts/${workoutId}/replies`,
      { params: httpParams },
    );
  }

  public downloadWorkout(id: number): Observable<HttpResponse<Blob>> {
    return this.http.get(`${this.baseUrl}/workouts/${id}/download`, {
      observe: 'response',
      responseType: 'blob',
    });
  }
}
