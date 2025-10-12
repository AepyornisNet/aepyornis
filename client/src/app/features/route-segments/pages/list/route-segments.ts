import { Component, inject, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { Api } from '../../../../core/services/api';
import { RouteSegment } from '../../../../core/types/route-segment';
import { PaginationParams } from '../../../../core/types/api-response';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { PaginatedListView } from '../../../../core/components/paginated-list-view/paginated-list-view';
import { RouteSegmentActionsComponent } from '../../../route-segments/components/route-segment-actions/route-segment-actions';

@Component({
  selector: 'app-route-segments',
  imports: [CommonModule, RouterLink, AppIcon, RouteSegmentActionsComponent],
  templateUrl: './route-segments.html'
})
export class RouteSegments extends PaginatedListView<RouteSegment> {
  private api = inject(Api);

  // Alias for better template readability
  routeSegments = this.items;
  hasRouteSegments = computed(() => this.hasItems());

  async loadData(page?: number) {
    if (page) {
      this.currentPage.set(page);
    }

    this.loading.set(true);
    this.error.set(null);

    const params: PaginationParams = {
      page: this.currentPage(),
      per_page: this.perPage()
    };

    try {
      const response = await firstValueFrom(this.api.getRouteSegments(params));

      if (response) {
        this.updatePaginationState(response);
      }
    } catch (err) {
      console.error('Failed to load route segments:', err);
      this.error.set('Failed to load route segments. Please try again.');
    } finally {
      this.loading.set(false);
    }
  }

  formatDate(dateString: string): string {
    return new Date(dateString).toLocaleDateString();
  }

  formatDistance(distance: number): string {
    return (distance / 1000).toFixed(2);
  }
}
