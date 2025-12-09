import {
  ChangeDetectionStrategy,
  Component,
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
  public readonly getRowLink = input.required<(item: T) => (string | number)[]>();

  public readonly selectionToggled = output<number | string>();

  // Content projection for custom templates
  public readonly headerTemplate = contentChild<TemplateRef<unknown>>('tableHeader');
  public readonly rowTemplate = contentChild.required<
    TemplateRef<{
      $implicit: T;
      index: number;
      multiSelectActive: boolean;
      onCellClick: (event: MouseEvent, item: T) => void;
      getRowLink: (item: T) => (string | number)[];
    }>
  >('tableRow');

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
