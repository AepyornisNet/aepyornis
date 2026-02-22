export type APIResponse<T = unknown> = {
  results: T;
  errors?: string[];
  error_codes?: string[];
};

export type PaginatedAPIResponse<T = unknown> = {
  results: T[];
  page: number;
  per_page: number;
  total_pages: number;
  total_count: number;
  errors?: string[];
  error_codes?: string[];
};

export type PaginationParams = {
  page?: number;
  per_page?: number;
};
