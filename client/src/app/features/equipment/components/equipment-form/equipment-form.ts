import { ChangeDetectionStrategy, Component, effect, input, output, signal } from '@angular/core';
import { form, FormField, FormRoot } from '@angular/forms/signals';
import { TranslatePipe } from '@ngx-translate/core';
import { Equipment } from '../../../../core/types/equipment';
import { WORKOUT_TYPES } from '../../../../core/types/workout-types';

export type EquipmentFormData = {
  name: string;
  description: string;
  notes: string;
  active: boolean;
  default_for: string[];
};

@Component({
  selector: 'app-equipment-form',
  imports: [FormField, FormRoot, TranslatePipe],
  templateUrl: './equipment-form.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class EquipmentForm {
  /** Optional equipment model to prefill the form for editing */
  public readonly equipment = input<Equipment | null>(null);

  /** Form submit output emitting the EquipmentFormData */
  public readonly formSubmit = output<EquipmentFormData>();

  public readonly workoutTypes = WORKOUT_TYPES;

  // Signal model representing form state
  public readonly equipmentModel = signal<EquipmentFormData>({
    name: '',
    description: '',
    notes: '',
    active: true,
    default_for: [],
  });

  public readonly equipmentSignalForm = form(this.equipmentModel, () => undefined, {
    submission: {
      action: async () => {
        this.formSubmit.emit(this.equipmentModel());
        return undefined;
      },
    },
  });

  public constructor() {
    effect(() => {
      const eq = this.equipment();
      if (eq) {
        this.equipmentModel.set({
          name: eq.name || '',
          description: eq.description || '',
          notes: eq.notes || '',
          active: eq.active ?? true,
          default_for: eq.default_for ? [...eq.default_for] : [],
        });
      } else {
        this.equipmentModel.set({
          name: '',
          description: '',
          notes: '',
          active: true,
          default_for: [],
        });
      }
    });
  }

  public isDefaultForSelected(value: string): boolean {
    return this.equipmentModel().default_for.includes(value);
  }

  public toggleDefaultFor(value: string): void {
    this.equipmentModel.update((prev) => {
      const next = new Set(prev.default_for);
      if (next.has(value)) {
        next.delete(value);
      } else {
        next.add(value);
      }
      return { ...prev, default_for: Array.from(next) };
    });
  }

  public onSubmit(event?: Event): void {
    if (event) {
      event.preventDefault();
    }
    this.formSubmit.emit(this.equipmentModel());
  }
}
