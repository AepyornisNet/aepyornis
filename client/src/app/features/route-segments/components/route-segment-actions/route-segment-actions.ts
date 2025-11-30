import { ChangeDetectionStrategy, Component, inject, input, output } from '@angular/core';

import { TranslatePipe } from '@ngx-translate/core';
import { Router } from '@angular/router';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { Api } from '../../../../core/services/api';
import { RouteSegment, RouteSegmentDetail } from '../../../../core/types/route-segment';

@Component({
  selector: 'app-route-segment-actions',
  imports: [AppIcon, TranslatePipe],
  templateUrl: './route-segment-actions.html',
  styleUrl: './route-segment-actions.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class RouteSegmentActionsComponent {
  public readonly routeSegment = input.required<RouteSegment | RouteSegmentDetail>();
  public readonly compact = input<boolean>(false);

  public readonly routeSegmentUpdated = output<void>();
  public readonly routeSegmentDeleted = output<void>();

  private api = inject(Api);
  private router = inject(Router);

  public showDeleteConfirm = false;
  public isProcessing = false;
  public errorMessage: string | null = null;
  public successMessage: string | null = null;

  public download(): void {
    if (this.isProcessing) {
      return;
    }

    this.isProcessing = true;
    this.errorMessage = null;

    this.api.downloadRouteSegment(this.routeSegment().id).subscribe({
      next: (blob) => {
        this.isProcessing = false;

        // Create download link
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;

        // Try to get filename from route segment
        const filename =
          (this.routeSegment() as RouteSegmentDetail).filename ||
          `route_segment_${this.routeSegment().id}.gpx`;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        window.URL.revokeObjectURL(url);
        document.body.removeChild(a);

        this.successMessage = 'Download started';
        setTimeout(() => (this.successMessage = null), 3000);
      },
      error: (err) => {
        this.isProcessing = false;
        this.errorMessage = 'Failed to download: ' + (err.error?.errors?.[0] || err.message);
        setTimeout(() => (this.errorMessage = null), 5000);
      },
    });
  }

  public edit(): void {
    this.router.navigate(['/route-segments', this.routeSegment().id, 'edit']);
  }

  public refresh(): void {
    if (this.isProcessing) {
      return;
    }

    this.isProcessing = true;
    this.errorMessage = null;

    this.api.refreshRouteSegment(this.routeSegment().id).subscribe({
      next: (response) => {
        this.isProcessing = false;
        this.successMessage = response.results.message;
        this.routeSegmentUpdated.emit();
        setTimeout(() => (this.successMessage = null), 3000);
      },
      error: (err) => {
        this.isProcessing = false;
        this.errorMessage = 'Failed to refresh: ' + (err.error?.errors?.[0] || err.message);
        setTimeout(() => (this.errorMessage = null), 5000);
      },
    });
  }

  public findMatches(): void {
    if (this.isProcessing) {
      return;
    }

    this.isProcessing = true;
    this.errorMessage = null;

    this.api.findRouteSegmentMatches(this.routeSegment().id).subscribe({
      next: (response) => {
        this.isProcessing = false;
        this.successMessage = response.results.message;
        this.routeSegmentUpdated.emit();
        setTimeout(() => (this.successMessage = null), 3000);
      },
      error: (err) => {
        this.isProcessing = false;
        this.errorMessage = 'Failed to find matches: ' + (err.error?.errors?.[0] || err.message);
        setTimeout(() => (this.errorMessage = null), 5000);
      },
    });
  }

  public confirmDelete(): void {
    this.showDeleteConfirm = true;
  }

  public cancelDelete(): void {
    this.showDeleteConfirm = false;
  }

  public delete(): void {
    if (this.isProcessing) {
      return;
    }

    this.isProcessing = true;
    this.errorMessage = null;

    this.api.deleteRouteSegment(this.routeSegment().id).subscribe({
      next: () => {
        this.isProcessing = false;
        this.showDeleteConfirm = false;
        this.routeSegmentDeleted.emit();
        this.router.navigate(['/route-segments']);
      },
      error: (err) => {
        this.isProcessing = false;
        this.errorMessage = 'Failed to delete: ' + (err.error?.errors?.[0] || err.message);
        setTimeout(() => (this.errorMessage = null), 5000);
      },
    });
  }
}
