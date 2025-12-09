import { ChangeDetectionStrategy, Component, computed, input, output, signal } from '@angular/core';
import { AppIcon } from '../app-icon/app-icon';
import { Pagination } from '../pagination/pagination';
import { TranslatePipe } from '@ngx-translate/core';

export type BaseListConfig = {
  title: string;
  addButtonText?: string;
  addButtonRoute?: string;
  enableSearch?: boolean;
  enableFilters?: boolean;
  enableMultiSelect?: boolean;
  searchPlaceholder?: string;
}

export type PaginationSource = {
  current: () => number;
  total: () => number;
  pages: () => number[];
  hasPrevious: () => boolean;
  hasNext: () => boolean;
  totalCount: () => number;
  previous: () => void;
  goTo: (page: number) => void;
  next: () => void;
}

@Component({
  selector: 'app-base-list',
  templateUrl: './base-list.html',
  styleUrl: './base-list.scss',
  imports: [AppIcon, Pagination, TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class BaseList {
  public readonly config = input.required<BaseListConfig>();
  public readonly loading = input<boolean>(false);
  public readonly error = input<string | null>(null);
  public readonly paginationSource = input.required<PaginationSource>();
  public readonly hasItems = input<boolean>(false);
  public readonly emptyMessage = input<string>('shared.no_items');

  public readonly addClicked = output<void>();
  public readonly searchChanged = output<string>();
  public readonly filtersToggled = output<void>();
  public readonly selectionChanged = output<Set<number | string>>();

  public readonly searchQuery = signal('');
  public readonly showFilters = signal(false);
  public readonly multiSelectActive = signal(false);
  public readonly selectedItems = signal<Set<number | string>>(new Set());

  public readonly hasSelection = computed(() => this.selectedItems().size > 0);
  public readonly selectionCount = computed(() => this.selectedItems().size);

  public onAddClick(): void {
    this.addClicked.emit();
  }

  public onSearchInput(event: Event): void {
    const value = (event.target as HTMLInputElement).value;
    this.searchQuery.set(value);
    this.searchChanged.emit(value);
  }

  public toggleFilters(): void {
    this.showFilters.update((v) => !v);
    this.filtersToggled.emit();
  }

  public toggleMultiSelect(): void {
    this.multiSelectActive.update((v) => !v);
    if (!this.multiSelectActive()) {
      this.clearSelection();
    }
  }

  public toggleItemSelection(id: number | string): void {
    this.selectedItems.update((selected) => {
      const newSet = new Set(selected);
      if (newSet.has(id)) {
        newSet.delete(id);
      } else {
        newSet.add(id);
      }
      return newSet;
    });
    this.selectionChanged.emit(this.selectedItems());
  }

  public isItemSelected(id: number | string): boolean {
    return this.selectedItems().has(id);
  }

  public clearSelection(): void {
    this.selectedItems.set(new Set());
    this.selectionChanged.emit(this.selectedItems());
  }

  public selectAll(ids: (number | string)[]): void {
    this.selectedItems.set(new Set(ids));
    this.selectionChanged.emit(this.selectedItems());
  }
}
