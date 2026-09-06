import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  OnInit,
  signal,
} from '@angular/core';
import { form, FormField, FormRoot } from '@angular/forms/signals';
import { Router } from '@angular/router';
import { firstValueFrom } from 'rxjs';
import { Api } from '../../../../core/services/api';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import {
  FileUploadList,
  UploadFileItem,
} from '../../../../core/components/file-upload-list/file-upload-list';
import { TranslatePipe, TranslateService } from '@ngx-translate/core';
import { RouteSegment, RouteSegmentDifficulty } from '../../../../core/types/route-segment';
import { WORKOUT_TYPES } from '../../../../core/types/workout-types';
import { getSportLabel } from '../../../../core/i18n/sport-labels';

@Component({
  selector: 'app-create-route-segment',
  imports: [FormField, FormRoot, AppIcon, FileUploadList, TranslatePipe],
  templateUrl: './create-route-segment.html',
  styleUrl: './create-route-segment.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class CreateRouteSegmentPage implements OnInit {
  public readonly sportLabel = getSportLabel;
  private api = inject(Api);
  private router = inject(Router);
  private translate = inject(TranslateService);

  public readonly uploadFiles = signal<UploadFileItem[]>([]);
  public readonly creating = signal(false);
  public readonly error = signal<string | null>(null);

  public readonly availableTypes = signal<string[]>([]);

  public readonly routeSegmentModel = signal({
    category: '',
    difficulty: '' as RouteSegmentDifficulty,
    visibility: 'public' as 'public' | 'followers' | '' | 'private',
    description: '',
    bidirectional: false,
    circular: false,
    notes: '',
  });

  public readonly routeSegmentForm = form(this.routeSegmentModel, {
    submission: {
      action: () => this.createRouteSegment(),
    },
  });

  public readonly hasFiles = computed(() => this.uploadFiles().length > 0);
  public readonly fileCount = computed(() => this.uploadFiles().length);

  public ngOnInit(): void {
    this.loadFilterOptions();
  }

  private async loadFilterOptions(): Promise<void> {
    const typesSet = new Set<string>();
    WORKOUT_TYPES.forEach((t) => {
      if (t.value !== 'all' && t.value !== 'auto') {
        typesSet.add(t.value);
      }
    });

    try {
      const res = await firstValueFrom(this.api.getWorkoutFilterOptions());
      if (res?.results?.types?.length) {
        res.results.types.forEach((t) => typesSet.add(t));
      }
    } catch (err) {
      console.error('Failed to load filter options:', err);
    }

    this.availableTypes.set(Array.from(typesSet));
  }

  public async createRouteSegment(): Promise<void> {
    if (this.creating()) {
      return;
    }

    const files = this.uploadFiles();
    if (files.length === 0) {
      this.error.set(this.translate.instant('Please select at least one file'));
      return;
    }

    this.creating.set(true);
    this.error.set(null);

    try {
      const formData = new FormData();
      files.forEach((item) => {
        formData.append('file', item.file);
        if (item.name.trim()) {
          formData.append('name', item.name.trim());
        } else {
          formData.append('name', '');
        }
      });

      const formValue = this.routeSegmentModel();
      const categoryValue = String(formValue.category || '').trim();
      if (categoryValue.length > 0) {
        formData.append('category', categoryValue);
      }

      if (formValue.difficulty) {
        formData.append('difficulty', formValue.difficulty);
      }

      if (formValue.visibility) {
        formData.append('visibility', formValue.visibility);
      }

      const descriptionValue = String(formValue.description || '').trim();
      if (descriptionValue.length > 0) {
        formData.append('description', descriptionValue);
      }

      const notesValue = String(formValue.notes || '').trim();
      if (notesValue.length > 0) {
        formData.append('notes', notesValue);
      }

      if (formValue.bidirectional) {
        formData.append('bidirectional', 'true');
      }
      if (formValue.circular) {
        formData.append('circular', 'true');
      }

      const response = await firstValueFrom(this.api.createRouteSegment(formData));
      const results = Array.isArray(response?.results) ? (response?.results as RouteSegment[]) : [];

      if (!results.length) {
        if (response?.errors?.length) {
          this.error.set(response.errors.join(' '));
        } else {
          this.error.set(
            this.translate.instant('Failed to create route segment. Please try again.'),
          );
        }
        return;
      }

      if (results.length === 1) {
        this.router.navigate(['/route-segments', results[0].id]);
      } else {
        this.router.navigate(['/route-segments']);
      }
    } catch (err) {
      console.error('Failed to create route segment:', err);
      this.error.set(this.translate.instant('Failed to create route segment. Please try again.'));
    } finally {
      this.creating.set(false);
    }
  }

  public goBack(): void {
    this.router.navigate(['/route-segments']);
  }
}
