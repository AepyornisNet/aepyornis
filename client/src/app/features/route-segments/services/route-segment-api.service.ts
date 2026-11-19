import { inject, Injectable } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';
import {
  APIResponse,
  PaginatedAPIResponse,
  PaginationParams,
} from '../../../core/types/api-response';
import {
  RouteSegment,
  RouteSegmentDetail,
  RouteSegmentDifficulty,
  RouteSegmentMatch,
} from '../../../core/types/route-segment';
import { Like } from '../../../core/types/like';

@Injectable({
  providedIn: 'root',
})
export class RouteSegmentApiService {
  private http = inject(HttpClient);
  private baseUrl = '/api/v2';

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
      category?: string;
      visibility?: 'public' | 'followers' | '' | 'private';
      description?: string;
      difficulty?: RouteSegmentDifficulty;
      notes?: string;
      bidirectional?: boolean;
      circular?: boolean;
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
      notes?: string;
      category?: string;
      visibility?: 'public' | 'followers' | '' | 'private';
      description?: string;
      difficulty?: RouteSegmentDifficulty;
      bidirectional?: boolean;
      circular?: boolean;
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

  public getRouteSegmentMatches(
    id: number,
    params?: { page?: number; per_page?: number; sort?: string },
  ): Observable<PaginatedAPIResponse<RouteSegmentMatch>> {
    let httpParams = new HttpParams();
    if (params?.page) {
      httpParams = httpParams.set('page', params.page.toString());
    }
    if (params?.per_page) {
      httpParams = httpParams.set('per_page', params.per_page.toString());
    }
    if (params?.sort) {
      httpParams = httpParams.set('sort', params.sort);
    }
    return this.http.get<PaginatedAPIResponse<RouteSegmentMatch>>(
      `${this.baseUrl}/route-segments/${id}/matches`,
      { params: httpParams },
    );
  }

  public likeRouteSegment(
    id: number,
  ): Observable<APIResponse<{ liked: boolean; like_count: number }>> {
    return this.http.post<APIResponse<{ liked: boolean; like_count: number }>>(
      `${this.baseUrl}/route-segments/${id}/like`,
      {},
    );
  }

  public unlikeRouteSegment(
    id: number,
  ): Observable<APIResponse<{ liked: boolean; like_count: number }>> {
    return this.http.delete<APIResponse<{ liked: boolean; like_count: number }>>(
      `${this.baseUrl}/route-segments/${id}/like`,
    );
  }

  public getRouteSegmentLikes(id: number): Observable<APIResponse<Like[]>> {
    return this.http.get<APIResponse<Like[]>>(`${this.baseUrl}/route-segments/${id}/likes`);
  }
}
