import { ChangeDetectionStrategy, Component, computed, inject, OnInit, signal } from '@angular/core';

import { FormBuilder, FormGroup, ReactiveFormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { Api } from '../../../../core/services/api';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { TranslatePipe } from '@ngx-translate/core';
import { RouteSegment } from '../../../../core/types/route-segment';

@Component({
  selector: 'app-create-route-segment',
  imports: [ReactiveFormsModule, AppIcon, TranslatePipe],
  templateUrl: './create-route-segment.html',
  styleUrl: './create-route-segment.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class CreateRouteSegmentPage implements OnInit {
  private api = inject(Api);
  private router = inject(Router);
  private fb = inject(FormBuilder);

  public readonly selectedFiles = signal<File[]>([]);
  public readonly creating = signal(false);
  public readonly error = signal<string | null>(null);
  public readonly bidirectional = signal(false);
  public readonly circular = signal(false);

  public routeSegmentForm!: FormGroup;

  public readonly hasFiles = computed(() => this.selectedFiles().length > 0);
  public readonly fileCount = computed(() => this.selectedFiles().length);

  public ngOnInit(): void {
    this.routeSegmentForm = this.fb.group({
      notes: [''],
    });
  }

  public onFilesSelected(event: Event): void {
    const input = event.target as HTMLInputElement;
    if (input.files) {
      this.selectedFiles.set(Array.from(input.files));
    }
  }

  public removeFile(index: number): void {
    const files = this.selectedFiles();
    files.splice(index, 1);
    this.selectedFiles.set([...files]);
  }

  public async createRouteSegment(): Promise<void> {
    if (this.creating()) {
      return;
    }

    const files = this.selectedFiles();
    if (files.length === 0) {
      this.error.set('Please select at least one file.');
      return;
    }

    this.creating.set(true);
    this.error.set(null);

    try {
      const formData = new FormData();
      files.forEach((file) => formData.append('file', file));

      const notesValue = String(this.routeSegmentForm.value.notes || '').trim();
      if (notesValue.length > 0) {
        formData.append('notes', notesValue);
      }

      const response = await firstValueFrom(this.api.createRouteSegment(formData));
      const results = Array.isArray(response?.results) ? (response?.results as RouteSegment[]) : [];

      if (!results.length) {
        if (response?.errors?.length) {
          this.error.set(response.errors.join(' '));
        } else {
          this.error.set('Failed to create route segment. Please try again.');
        }
        return;
      }

      if (this.bidirectional() || this.circular()) {
        for (const segment of results) {
          await firstValueFrom(
            this.api.updateRouteSegment(segment.id, {
              name: segment.name,
              notes: segment.notes ?? notesValue,
              bidirectional: this.bidirectional(),
              circular: this.circular(),
            }),
          );
        }
      }

      if (results.length === 1) {
        this.router.navigate(['/route-segments', results[0].id]);
      } else {
        this.router.navigate(['/route-segments']);
      }
    } catch (err) {
      console.error('Failed to create route segment:', err);
      this.error.set('Failed to create route segment. Please try again.');
    } finally {
      this.creating.set(false);
    }
  }

  public goBack(): void {
    this.router.navigate(['/route-segments']);
  }
}
