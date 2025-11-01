import { Component, computed, input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { TranslatePipe } from '@ngx-translate/core';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { Totals } from '../../../../core/types/workout';

@Component({
  selector: 'app-key-metrics',
  imports: [CommonModule, AppIcon, TranslatePipe],
  templateUrl: './key-metrics.html',
  styleUrl: './key-metrics.scss',
})
export class KeyMetrics {
  readonly totals = input<Totals | null>(null);

  readonly totalWorkoutsCount = computed(() => this.totals()?.workouts || 0);
  readonly totalDistance = computed(() => {
    const distance = this.totals()?.distance || 0;
    return (distance / 1000).toFixed(2); // Convert to km
  });
  readonly totalDuration = computed(() => {
    const duration = this.totals()?.duration || 0;
    const hours = Math.floor(duration / 3600);
    const minutes = Math.floor((duration % 3600) / 60);
    return `${hours}h ${minutes}m`;
  });
  readonly totalElevation = computed(() => {
    const up = this.totals()?.up || 0;
    return up.toFixed(0);
  });
}
