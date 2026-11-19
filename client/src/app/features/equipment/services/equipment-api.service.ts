import { inject, Injectable } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';
import {
  APIResponse,
  PaginatedAPIResponse,
  PaginationParams,
} from '../../../core/types/api-response';
import { Equipment } from '../../../core/types/equipment';

@Injectable({
  providedIn: 'root',
})
export class EquipmentApiService {
  private http = inject(HttpClient);
  private baseUrl = '/api/v2';

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
}
