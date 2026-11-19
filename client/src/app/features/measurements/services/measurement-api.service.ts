import { inject, Injectable } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';
import {
  APIResponse,
  PaginatedAPIResponse,
  PaginationParams,
} from '../../../core/types/api-response';
import { Measurement } from '../../../core/types/measurement';

@Injectable({
  providedIn: 'root',
})
export class MeasurementApiService {
  private http = inject(HttpClient);
  private baseUrl = '/api/v2';

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
}
