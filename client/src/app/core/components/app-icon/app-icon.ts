import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { NgIconComponent } from '@ng-icons/core';
import { getIcon } from '../../types/icon-map';

/**
 * AppIcon Component
 *
 * Wraps ng-icon and provides a consistent way to use icons throughout the application.
 * Uses the icon-map.ts to resolve icon keys to their corresponding ng-icons identifiers.
 *
 * Usage:
 * <app-icon name="workout" />
 * <app-icon name="dashboard" size="24" />
 * <app-icon name="edit" size="1.5rem" class="text-primary" />
 */
@Component({
  selector: 'app-icon',
  imports: [NgIconComponent],
  template: `
    <ng-icon [name]="iconName()" [size]="sizeAsString()" [strokeWidth]="strokeWidthAsString()" />
  `,
  styles: [
    `
      :host {
        display: inline-flex;
        align-items: center;
        justify-content: center;
      }
      ng-icon {
        display: inline-flex;
      }
    `,
  ],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AppIcon {
  /**
   * Icon key from the icon map (e.g., 'workout', 'dashboard', 'edit')
   */
  readonly name = input.required<string>();

  /**
   * Icon size (e.g., '24', '1.5rem', '48')
   */
  readonly size = input<string | number>();

  /**
   * Stroke width for icons that support it
   */
  readonly strokeWidth = input<string | number>();

  /**
   * Resolved icon name from the icon map
   */
  iconName = () => getIcon(this.name());

  /**
   * Convert size to string for ng-icon
   */
  sizeAsString = (): string => {
    const sizeValue = this.size();
    return sizeValue ? String(sizeValue) : '';
  };

  /**
   * Convert strokeWidth to string for ng-icon
   */
  strokeWidthAsString = (): string => {
    const strokeValue = this.strokeWidth();
    return strokeValue ? String(strokeValue) : '';
  };
}
