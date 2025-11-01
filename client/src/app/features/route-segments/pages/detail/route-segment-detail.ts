import { Component, inject, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { Api } from '../../../../core/services/api';
import { RouteSegmentDetail } from '../../../../core/types/route-segment';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { RouteSegmentActionsComponent } from '../../../route-segments/components/route-segment-actions/route-segment-actions';
import { TranslatePipe } from '@ngx-translate/core';

@Component({
  selector: 'app-route-segment-detail',
  imports: [CommonModule, RouterLink, AppIcon, RouteSegmentActionsComponent, TranslatePipe],
  templateUrl: './route-segment-detail.html',
})
export class RouteSegmentDetailPage implements OnInit {
  private api = inject(Api);
  private route = inject(ActivatedRoute);
  private router = inject(Router);

  readonly routeSegment = signal<RouteSegmentDetail | null>(null);
  readonly loading = signal(true);
  readonly error = signal<string | null>(null);

  ngOnInit() {
    this.route.params.subscribe((params) => {
      const id = parseInt(params['id']);
      if (id) {
        this.loadRouteSegment(id);
      }
    });
  }

  async loadRouteSegment(id: number) {
    this.loading.set(true);
    this.error.set(null);

    try {
      const response = await firstValueFrom(this.api.getRouteSegment(id));

      if (response) {
        this.routeSegment.set(response.results);
      }
    } catch (err) {
      console.error('Failed to load route segment:', err);
      this.error.set('Failed to load route segment. Please try again.');
    } finally {
      this.loading.set(false);
    }
  }

  onRouteSegmentUpdated() {
    // Reload the route segment to get the updated state
    const id = this.route.snapshot.params['id'];
    if (id) {
      this.loadRouteSegment(parseInt(id));
    }
  }

  onRouteSegmentDeleted() {
    // Navigation is handled by the actions component
  }

  formatDate(dateString: string): string {
    return new Date(dateString).toLocaleDateString();
  }

  formatDistance(distance: number): string {
    return (distance / 1000).toFixed(2);
  }

  formatDuration(seconds: number): string {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const secs = seconds % 60;

    if (hours > 0) {
      return `${hours}:${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
    }
    return `${minutes}:${secs.toString().padStart(2, '0')}`;
  }

  formatSpeed(speedMs: number): string {
    return (speedMs * 3.6).toFixed(2);
  }

  formatTempo(speedMs: number): string {
    if (speedMs === 0) {
      return '-';
    }
    const tempoSecondsPerKm = 1000 / speedMs;
    const minutes = Math.floor(tempoSecondsPerKm / 60);
    const seconds = Math.floor(tempoSecondsPerKm % 60);
    return `${minutes}:${seconds.toString().padStart(2, '0')}`;
  }

  goBack() {
    this.router.navigate(['/route-segments']);
  }
}
