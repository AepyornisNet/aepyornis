import { Component, OnInit, signal, inject, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { firstValueFrom } from 'rxjs';
import { Api } from '../../../../core/services/api';
import { WorkoutDetail } from '../../../../core/types/workout';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';

@Component({
  selector: 'app-create-route-segment',
  imports: [CommonModule, FormsModule, AppIcon],
  templateUrl: './create-route-segment.html',
  styleUrl: './create-route-segment.scss'
})
export class CreateRouteSegmentPage implements OnInit {
  private api = inject(Api);
  private route = inject(ActivatedRoute);
  private router = inject(Router);

  workout = signal<WorkoutDetail | null>(null);
  loading = signal(true);
  error = signal<string | null>(null);
  creating = signal(false);

  // Form fields
  name = signal('');
  start = signal(1);
  end = signal(1);

  // Computed values
  totalPoints = computed(() => {
    const w = this.workout();
    return w?.map_data?.details?.position?.length || 0;
  });

  selectedDistance = computed(() => {
    const w = this.workout();
    const startIdx = this.start() - 1;
    const endIdx = this.end() - 1;
    
    if (!w?.map_data?.details?.distance || startIdx < 0 || endIdx < 0) {
      return 0;
    }

    const distances = w.map_data.details.distance;
    if (endIdx >= distances.length || startIdx >= distances.length) {
      return 0;
    }

    return Math.abs(distances[endIdx] - distances[startIdx]);
  });

  ngOnInit() {
    this.route.params.subscribe(params => {
      const id = parseInt(params['id']);
      if (id) {
        this.loadWorkout(id);
      }
    });
  }

  async loadWorkout(id: number) {
    this.loading.set(true);
    this.error.set(null);

    try {
      const response = await firstValueFrom(this.api.getWorkout(id));

      if (response) {
        const workout = response.results;
        this.workout.set(workout);
        this.name.set(workout.name);
        
        // Set end to the last point
        const points = workout.map_data?.details?.position?.length || 1;
        this.end.set(points);
      }
    } catch (err) {
      console.error('Failed to load workout:', err);
      this.error.set('Failed to load workout. Please try again.');
    } finally {
      this.loading.set(false);
    }
  }

  updateStart(value: number) {
    this.start.set(value);
    
    // Ensure start is not greater than end
    if (value > this.end()) {
      this.end.set(value);
    }
  }

  updateEnd(value: number) {
    this.end.set(value);
    
    // Ensure end is not less than start
    if (value < this.start()) {
      this.start.set(value);
    }
  }

  async createRouteSegment() {
    if (this.creating()) return;

    const w = this.workout();
    if (!w) return;

    this.creating.set(true);
    this.error.set(null);

    try {
      const response = await firstValueFrom(this.api.createRouteSegmentFromWorkout(w.id, {
        name: this.name(),
        start: this.start(),
        end: this.end()
      }));

      if (response) {
        // Navigate to the created route segment
        this.router.navigate(['/route-segments', response.results.id]);
      }
    } catch (err) {
      console.error('Failed to create route segment:', err);
      this.error.set('Failed to create route segment. Please try again.');
      this.creating.set(false);
    }
  }

  goBack() {
    const w = this.workout();
    if (w) {
      this.router.navigate(['/workouts', w.id]);
    } else {
      this.router.navigate(['/workouts']);
    }
  }
}
