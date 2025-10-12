import { Component, input, output, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { Api } from '../../../../core/services/api';
import { RouteSegment, RouteSegmentDetail } from '../../../../core/types/route-segment';

@Component({
  selector: 'app-route-segment-actions',
  imports: [CommonModule, AppIcon],
  templateUrl: './route-segment-actions.html',
  styleUrl: './route-segment-actions.scss'
})
export class RouteSegmentActionsComponent {
  routeSegment = input.required<RouteSegment | RouteSegmentDetail>();
  compact = input<boolean>(false);
  
  routeSegmentUpdated = output<void>();
  routeSegmentDeleted = output<void>();
  
  private api = inject(Api);
  private router = inject(Router);
  
  showDeleteConfirm = false;
  isProcessing = false;
  errorMessage: string | null = null;
  successMessage: string | null = null;

  download() {
    if (this.isProcessing) return;
    
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
        const filename = (this.routeSegment() as RouteSegmentDetail).filename || 
                        `route_segment_${this.routeSegment().id}.gpx`;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        window.URL.revokeObjectURL(url);
        document.body.removeChild(a);
        
        this.successMessage = 'Download started';
        setTimeout(() => this.successMessage = null, 3000);
      },
      error: (err) => {
        this.isProcessing = false;
        this.errorMessage = 'Failed to download: ' + (err.error?.errors?.[0] || err.message);
        setTimeout(() => this.errorMessage = null, 5000);
      }
    });
  }

  edit() {
    this.router.navigate(['/route-segments', this.routeSegment().id, 'edit']);
  }

  refresh() {
    if (this.isProcessing) return;
    
    this.isProcessing = true;
    this.errorMessage = null;
    
    this.api.refreshRouteSegment(this.routeSegment().id).subscribe({
      next: (response) => {
        this.isProcessing = false;
        this.successMessage = response.results.message;
        this.routeSegmentUpdated.emit();
        setTimeout(() => this.successMessage = null, 3000);
      },
      error: (err) => {
        this.isProcessing = false;
        this.errorMessage = 'Failed to refresh: ' + (err.error?.errors?.[0] || err.message);
        setTimeout(() => this.errorMessage = null, 5000);
      }
    });
  }

  findMatches() {
    if (this.isProcessing) return;
    
    this.isProcessing = true;
    this.errorMessage = null;
    
    this.api.findRouteSegmentMatches(this.routeSegment().id).subscribe({
      next: (response) => {
        this.isProcessing = false;
        this.successMessage = response.results.message;
        this.routeSegmentUpdated.emit();
        setTimeout(() => this.successMessage = null, 3000);
      },
      error: (err) => {
        this.isProcessing = false;
        this.errorMessage = 'Failed to find matches: ' + (err.error?.errors?.[0] || err.message);
        setTimeout(() => this.errorMessage = null, 5000);
      }
    });
  }

  confirmDelete() {
    this.showDeleteConfirm = true;
  }

  cancelDelete() {
    this.showDeleteConfirm = false;
  }

  delete() {
    if (this.isProcessing) return;
    
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
        setTimeout(() => this.errorMessage = null, 5000);
      }
    });
  }
}
