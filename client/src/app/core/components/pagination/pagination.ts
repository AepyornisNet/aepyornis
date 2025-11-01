import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { TranslatePipe } from '@ngx-translate/core';

@Component({
  selector: 'app-pagination',
  templateUrl: './pagination.html',
  imports: [TranslatePipe],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Pagination {
  public readonly source = input.required<{
    current: () => number;
    total: () => number;
    pages: () => number[];
    hasPrevious: () => boolean;
    hasNext: () => boolean;
    totalCount: () => number;
    previous: () => void;
    goTo: (page: number) => void;
    next: () => void;
  }>();

  public getCurrent(): number {
    return this.source().current();
  }

  public getTotal(): number {
    return this.source().total();
  }

  public getPages(): number[] {
    return this.source().pages();
  }

  public getHasPrevious(): boolean {
    return this.source().hasPrevious();
  }

  public getHasNext(): boolean {
    return this.source().hasNext();
  }

  public getTotalCount(): number {
    return this.source().totalCount();
  }

  public onPrevious(): void {
    this.source().previous();
  }

  public onNext(): void {
    this.source().next();
  }

  public onGoTo(page: number): void {
    this.source().goTo(page);
  }
}
