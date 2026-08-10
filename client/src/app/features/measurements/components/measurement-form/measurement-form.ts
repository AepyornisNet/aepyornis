import { ChangeDetectionStrategy, Component, effect, input, output, signal } from '@angular/core';
import { form, FormField, FormRoot } from '@angular/forms/signals';
import { TranslatePipe } from '@ngx-translate/core';
import { Measurement } from '../../../../core/types/measurement';

export type MeasurementFormData = {
  date: string;
  weight: number | null;
  height: number | null;
  steps: number | null;
  ftp: number | null;
  resting_heart_rate: number | null;
  max_heart_rate: number | null;
};

@Component({
  selector: 'app-measurement-form',
  imports: [FormField, FormRoot, TranslatePipe],
  templateUrl: './measurement-form.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class MeasurementForm {
  /** Optional measurement model to prefill for editing */
  public readonly measurement = input<Measurement | null>(null);

  /** Form submit output emitting MeasurementFormData */
  public readonly formSubmit = output<MeasurementFormData>();

  public readonly measurementModel = signal<MeasurementFormData>({
    date: this.getTodayDate(),
    weight: null,
    height: null,
    steps: null,
    ftp: null,
    resting_heart_rate: null,
    max_heart_rate: null,
  });

  public readonly measurementSignalForm = form(this.measurementModel, () => undefined, {
    submission: {
      action: async () => {
        this.formSubmit.emit(this.measurementModel());
        return undefined;
      },
    },
  });

  public constructor() {
    effect(() => {
      const m = this.measurement();
      if (m) {
        this.measurementModel.set({
          date: this.formatDateForInput(m.date),
          weight: m.weight || null,
          height: m.height || null,
          steps: m.steps || null,
          ftp: m.ftp || null,
          resting_heart_rate: m.resting_heart_rate || null,
          max_heart_rate: m.max_heart_rate || null,
        });
      } else {
        this.measurementModel.set({
          date: this.getTodayDate(),
          weight: null,
          height: null,
          steps: null,
          ftp: null,
          resting_heart_rate: null,
          max_heart_rate: null,
        });
      }
    });
  }

  private getTodayDate(): string {
    const today = new Date();
    return today.toISOString().split('T')[0];
  }

  private formatDateForInput(dateString: string): string {
    if (!dateString) {
      return '';
    }
    return dateString.split('T')[0];
  }

  public onSubmit(event?: Event): void {
    if (event) {
      event.preventDefault();
    }
    this.formSubmit.emit(this.measurementModel());
  }
}
