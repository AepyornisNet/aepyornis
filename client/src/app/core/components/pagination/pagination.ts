import { Component, Input } from '@angular/core';
import { TranslatePipe } from '@ngx-translate/core';

@Component({
  selector: 'app-pagination',
  templateUrl: './pagination.html',
  imports: [TranslatePipe],
})
export class Pagination {
  @Input() source!: {
    current: () => number;
    total: () => number;
    pages: () => number[];
    hasPrevious: () => boolean;
    hasNext: () => boolean;
    totalCount: () => number;
    previous: () => void;
    goTo: (page: number) => void;
    next: () => void;
  };

  getCurrent() {
    return this.source.current();
  }
  getTotal() {
    return this.source.total();
  }
  getPages() {
    return this.source.pages();
  }
  getHasPrevious() {
    return this.source.hasPrevious();
  }
  getHasNext() {
    return this.source.hasNext();
  }
  getTotalCount() {
    return this.source.totalCount();
  }

  onPrevious() {
    this.source.previous();
  }

  onNext() {
    this.source.next();
  }

  onGoTo(page: number) {
    this.source.goTo(page);
  }
}
