import {
  ChangeDetectionStrategy,
  Component,
  computed,
  contentChild,
  input,
  output,
  TemplateRef,
  ViewEncapsulation,
} from '@angular/core';
import { NgTemplateOutlet } from '@angular/common';

@Component({
  selector: 'app-base-table',
  templateUrl: './base-table.html',
  styleUrl: './base-table.scss',
  imports: [NgTemplateOutlet],
  encapsulation: ViewEncapsulation.None,
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class BaseTable<T extends { id: number | string }> {
  public readonly items = input.required<T[]>();
  public readonly multiSelectActive = input<boolean>(false);
  public readonly selectedItems = input<Set<number | string>>(new Set());
  public readonly getRowLink = input<((item: T) => (string | number)[]) | null>(null);

  public readonly selectionToggled = output<number | string>();
  public readonly selectAllToggled = output<void>();

  /** 'none' | 'some' | 'all' — drives the header checkbox state, scoped to current page */
  public readonly selectionState = computed<'none' | 'some' | 'all'>(() => {
    const currentPageIds = this.items().map((item) => item.id);
    const total = currentPageIds.length;
    if (total === 0) {
      return 'none';
    }
    const selectedOnPage = currentPageIds.filter((id) => this.selectedItems().has(id)).length;
    if (selectedOnPage === 0) {
      return 'none';
    }
    if (selectedOnPage >= total) {
      return 'all';
    }
    return 'some';
  });

  // Content projection for custom templates
  public readonly headerTemplate = contentChild<TemplateRef<unknown>>('tableHeader');
  public readonly rowTemplate = contentChild.required<
    TemplateRef<{
      $implicit: T;
      index: number;
      multiSelectActive: boolean;
      onCellClick: (event: MouseEvent, item: T) => void;
      getRowLink: ((item: T) => (string | number)[]) | null;
    }>
  >('tableRow');
  public readonly mobileRowTemplate = contentChild<
    TemplateRef<{
      $implicit: T;
      index: number;
      multiSelectActive: boolean;
      isSelected: boolean;
      onCheckboxClick: (item: T) => void;
    }>
  >('mobileRow');

  public isSelected(id: number | string): boolean {
    return this.selectedItems().has(id);
  }

  public onCellClick(event: MouseEvent, item: T): void {
    if (this.multiSelectActive()) {
      event.preventDefault();
      this.selectionToggled.emit(item.id);
    }
  }

  public onCheckboxClick(item: T): void {
    this.selectionToggled.emit(item.id);
  }
}
