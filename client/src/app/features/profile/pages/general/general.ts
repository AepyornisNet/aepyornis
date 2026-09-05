import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { FormField, FormRoot } from '@angular/forms/signals';
import { TranslatePipe } from '@ngx-translate/core';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { ProfileStore } from '../../services/profile-store';
import { AVAILABLE_LANGUAGES } from '../../../../core/config/languages';

@Component({
  selector: 'app-profile-general',
  imports: [FormField, FormRoot, TranslatePipe, AppIcon],
  templateUrl: './general.html',
  styleUrl: './general.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ProfileGeneralPage {
  protected store = inject(ProfileStore);
  protected languages = AVAILABLE_LANGUAGES;
}
