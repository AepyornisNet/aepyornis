export interface APIResponse<T = unknown> {
  results: T;
  errors?: string[];
}

export interface PaginatedAPIResponse<T = unknown> {
  results: T[];
  page: number;
  per_page: number;
  total_pages: number;
  total_count: number;
  errors?: string[];
}

export interface PaginationParams {
  page?: number;
  per_page?: number;
}
