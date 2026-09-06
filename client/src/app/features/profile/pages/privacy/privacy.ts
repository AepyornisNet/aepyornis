import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { FormField, FormRoot } from '@angular/forms/signals';
import { TranslatePipe } from '@ngx-translate/core';
import { AppIcon } from '../../../../core/components/app-icon/app-icon';
import { ProfileStore } from '../../services/profile-store';

@Component({
  selector: 'app-profile-privacy',
  imports: [FormField, FormRoot, TranslatePipe, AppIcon],
  templateUrl: './privacy.html',
  styleUrl: './privacy.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ProfilePrivacyPage {
  protected store = inject(ProfileStore);
}
