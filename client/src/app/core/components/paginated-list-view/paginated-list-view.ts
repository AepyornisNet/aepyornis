import { computed, Directive, OnInit, signal } from '@angular/core';

/**
 * Abstract base class for paginated list views
 *
 * This class provides common functionality for components that display paginated lists:
 * - Pagination state management (current page, total pages, etc.)
 * - Loading and error states
 * - Navigation methods (next page, previous page, go to page)
 * - Computed values for pagination logic
 * - Visible page numbers calculation
 *
 * Usage:
 * 1. Extend this class in your component
 * 2. Implement the abstract loadData() method
 * 3. Call loadData() in ngOnInit()
 * 4. Use the provided signals and methods in your template
 */
@Directive()
export abstract class PaginatedListView<T> implements OnInit {
  // Data state
  public readonly items = signal<T[]>([]);
  public readonly loading = signal(true);
  public readonly error = signal<string | null>(null);

  // Pagination state
  public readonly currentPage = signal(1);
  public readonly perPage = signal(20);
  public readonly totalPages = signal(0);
  public readonly totalCount = signal(0);

  // Computed values
  public readonly hasItems = computed<boolean>(() => this.items().length > 0);
  public readonly hasPreviousPage = computed<boolean>(() => this.currentPage() > 1);
  public readonly hasNextPage = computed<boolean>(() => this.currentPage() < this.totalPages());

  /**
   * Calculate visible page numbers for pagination UI
   * Shows max 7 page buttons with ellipsis when needed
   */
  public readonly visiblePages = computed<number[]>(() => {
    const current = this.currentPage();
    const total = this.totalPages();
    const maxVisible = 7;

    if (total <= maxVisible) {
      return Array.from({ length: total }, (_, i) => i + 1);
    }

    const pages: number[] = [];
    pages.push(1);

    let start = Math.max(2, current - 2);
    let end = Math.min(total - 1, current + 2);

    if (current <= 3) {
      end = Math.min(total - 1, 5);
    }

    if (current >= total - 2) {
      start = Math.max(2, total - 4);
    }

    if (start > 2) {
      pages.push(-1); // -1 represents ellipsis
    }

    for (let i = start; i <= end; i++) {
      pages.push(i);
    }

    if (end < total - 1) {
      pages.push(-1);
    }

    if (total > 1) {
      pages.push(total);
    }

    return pages;
  });

  public ngOnInit(): void {
    this.loadData();
  }

  /**
   * Abstract method to load data from the API
   * Must be implemented by subclasses
   * @param page Optional page number to load
   */
  public abstract loadData(page?: number): Promise<void>;

  /**
   * Navigate to a specific page
   * @param page The page number to navigate to
   */
  public goToPage(page: number): void {
    this.loadData(page);
  }

  /**
   * Navigate to the previous page
   */
  public previousPage(): void {
    if (this.hasPreviousPage()) {
      this.loadData(this.currentPage() - 1);
    }
  }

  /**
   * Navigate to the next page
   */
  public nextPage(): void {
    if (this.hasNextPage()) {
      this.loadData(this.currentPage() + 1);
    }
  }

  /**
   * Helper method to update pagination state from API response
   * @param response The API response containing pagination data
   */
  protected updatePaginationState(response: {
    results: T[];
    page: number;
    per_page: number;
    total_pages: number;
    total_count: number;
  }): void {
    this.items.set(response.results);
    this.currentPage.set(response.page);
    this.perPage.set(response.per_page);
    this.totalPages.set(response.total_pages);
    this.totalCount.set(response.total_count);
  }

  /**
   * Provide a compact pagination context object for pagination components.
   * This allows templates to bind a single [source]="pagination()" instead of multiple attributes.
   */
  public pagination(): {
    current: () => number;
    total: () => number;
    pages: () => number[];
    hasPrevious: () => boolean;
    hasNext: () => boolean;
    totalCount: () => number;
    previous: () => void;
    goTo: (page: number) => void;
    next: () => void;
  } {
    return {
      // getters return the current values so templates can call them: source.current()
      current: (): number => this.currentPage(),
      total: (): number => this.totalPages(),
      pages: (): number[] => this.visiblePages(),
      hasPrevious: (): boolean => this.hasPreviousPage(),
      hasNext: (): boolean => this.hasNextPage(),
      totalCount: (): number => this.totalCount(),

      // navigation helpers call the existing methods on the PaginatedListView
      previous: (): void => this.previousPage(),
      goTo: (page: number): void => this.goToPage(page),
      next: (): void => this.nextPage(),
    };
  }
}
